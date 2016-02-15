package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ftwynn/gologgen/loggensender"

	log "github.com/Sirupsen/logrus"
)

var confPath string
var workers int

// GlobalConfStore holds all the config data from the conf file
type GlobalConfStore struct {
	HTTPLoc        string               `json:"httpLoc"`
	OutputType     string               `json:"OutputType"`
	SyslogType     string               `json:"SyslogType"`
	SyslogLoc      string               `json:"SyslogLoc"`
	FileOutputPath string               `json:"FileOutputPath"`
	DataFiles      []DataFileMetaData   `json:"DataFiles"`
	ReplayFiles    []ReplayFileMetaData `json:"ReplayFiles"`
	HTTPClient     http.Client
	FileHandler    *os.File
}

// DataFileMetaData stores the configs around data files
type DataFileMetaData struct {
	Path string `json:"Path"`
}

// ReplayFileMetaData stores all the info around a replay file
type ReplayFileMetaData struct {
	Path            string                           `json:"Path"`
	TimestampRegex  string                           `json:"TimestampRegex"`
	TimestampFormat string                           `json:"TimestampFormat"`
	RepeatInterval  int                              `json:"RepeatInterval"`
	Headers         []loggensender.LogLineHTTPHeader `json:"Headers"`
}

// LogGenDataFile represents a data file
type LogGenDataFile struct {
	Lines []loggensender.LogLineProperties `json:"lines"`
}

func init() {
	// Set global logging levels by the flag, default to WARN if not defined
	var level string
	flag.StringVar(&level, "level", "WARN", "Log level for the gologgen program itself")
	flag.StringVar(&confPath, "conf", "config/gologgen.conf", "Optional path for the config file")
	flag.IntVar(&workers, "workers", 10, "Number of workers to spawn for queue processing")

	flag.Parse()

	level = strings.ToUpper(level)

	log.SetFormatter(&log.TextFormatter{ForceColors: true})

	switch level {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

}

// InitializeRunTable will take a slice of LogLines and start times and put the various lines in their starting slots in the map
func queueLogLines(Lines []loggensender.LogLineProperties, tickerStart time.Time, runQueue chan loggensender.LogLineProperties) {
	for _, line := range Lines {
		log.Debug("========== New Line ==========")
		log.WithFields(log.Fields{
			"time": line.StartTime,
			"line": line,
		}).Debug("The literal time string")

		// Get the log line target start time
		var targetTime time.Time
		if line.StartTime == "" {
			targetTime = tickerStart
		} else {
			// Use regex to take in what the start time should be
			re := regexp.MustCompile(`\d+`)
			targetHourMinSec := re.FindAllString(line.StartTime, -1)
			targetHour, _ := strconv.Atoi(targetHourMinSec[0])
			targetMin, _ := strconv.Atoi(targetHourMinSec[1])
			targetSec, _ := strconv.Atoi(targetHourMinSec[2])
			loc, _ := time.LoadLocation("America/Los_Angeles")
			targetTime = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), targetHour, targetMin, targetSec, 0, loc).Truncate(time.Second)
		}

		log.WithFields(log.Fields{
			"targetTime": targetTime,
		}).Debug("The target time is translated to")

		log.WithFields(log.Fields{
			"tickerStart": tickerStart,
		}).Debug("The start time of the ticker is")

		diff := targetTime.Sub(tickerStart)

		log.WithFields(log.Fields{
			"diff": diff,
		}).Debug("The diff between target and tickerStart is")

		diffMod := int(math.Abs(float64(int(diff.Seconds()) % line.IntervalSecs)))

		switch {
		case targetTime.Equal(tickerStart) || targetTime.After(tickerStart):
			log.Debug("Target is equal to or after start, so queuing with target time")
			go sleepAndSend(runQueue, targetTime, line)
		case targetTime.Before(tickerStart):
			if diffMod == 0 {
				log.Debug("TickerStart is a multiple of Target's interval, so setting to TickerStart")
				go sleepAndSend(runQueue, tickerStart, line)
			} else {
				log.WithFields(log.Fields{
					"startTime": tickerStart.Add(time.Duration(diffMod) * time.Second),
				}).Debug("Setting a start after ticker start")
				go sleepAndSend(runQueue, tickerStart.Add(time.Duration(diffMod)*time.Second), line)
			}
		}

	}

}

func sleepAndSend(runQueue chan loggensender.LogLineProperties, targetTime time.Time, logline loggensender.LogLineProperties) {
	currentTime := time.Now()

	log.WithFields(log.Fields{
		"line":        logline.Text,
		"targetTime":  targetTime,
		"currentTime": currentTime,
	}).Debug("Starting new sleepAndSend")

	time.Sleep(targetTime.Sub(currentTime))

	log.WithFields(log.Fields{
		"line":       logline.Text,
		"targetTime": targetTime,
	}).Debug("Queuing line")

	runQueue <- logline

	// Calculate next run time
	// Randomize the Interval by specifying the std dev and adding the desired mean
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var milliseconds int
	if logline.IntervalMillis != 0 {
		milliseconds = logline.IntervalSecs
	} else {
		milliseconds = logline.IntervalSecs * 1000
	}
	var stdDevMilli int
	if logline.IntervalMillis != 0 {
		stdDevMilli = logline.IntervalStdDevMillis
	} else {
		stdDevMilli = int(logline.IntervalStdDev * 1000.0)
	}
	nextInterval := int(r.NormFloat64()*float64(stdDevMilli) + float64(milliseconds))
	nextTime := targetTime.Add(time.Duration(nextInterval) * time.Millisecond)
	log.WithFields(log.Fields{
		"line":     logline.Text,
		"nextTime": nextTime,
	}).Debug("SCHEDULED - Next log run")

	go sleepAndSend(runQueue, nextTime, logline)
}

// storeDataFileLogLines takes the conf data, gets the associated files, and puts them in a big list of LogLine Objects
func parseAndStoreLogLines(confData GlobalConfStore, targetStartTime time.Time) (logLines []loggensender.LogLineProperties) {
	log.WithFields(log.Fields{
		"confData": confData,
	}).Info("Entering parseAndQueueLogLines")

	dataJSON := LogGenDataFile{}

	// First, read in any data files
	for _, dataFile := range confData.DataFiles {
		dataText, err := ioutil.ReadFile(dataFile.Path)
		if err != nil {
			log.WithFields(log.Fields{
				"error_msg": err,
				"file_path": dataFile.Path,
			}).Error("Couldn't read in the data file, so ignoring and moving on to the next data file")
			continue
		}

		// Convert the data to something we can work with
		err = json.Unmarshal(dataText, &dataJSON)
		if err != nil {
			log.WithFields(log.Fields{
				"error_msg": err,
				"file_path": dataFile.Path,
			}).Error("Couldn't read in text from found data file, so ignoring and moving on to the next data file")
			continue
		}

		validateDataFile(&dataJSON)

		// Add the parsed fields to the queue
		for i := 0; i < len(dataJSON.Lines); i++ {
			logLines = append(logLines, dataJSON.Lines[i])
		}
	}

	// Second, read in the replay files
	for _, replayFile := range confData.ReplayFiles {

		file, err := os.Open(replayFile.Path)
		if err != nil {
			log.WithFields(log.Fields{
				"error_msg": err,
				"path":      replayFile.Path,
			}).Error("Something went amiss trying to read the replay file")
		}
		defer file.Close()

		// Set the timestamp regex
		var timeRegex = regexp.MustCompile(replayFile.TimestampRegex)

		// Scan the file through line by line
		scanner := bufio.NewScanner(file)
		log.WithFields(log.Fields{
			"path": replayFile.Path,
		}).Debug("Scanning replay file")
		for scanner.Scan() {
			line := scanner.Text()
			log.WithFields(log.Fields{
				"line": line,
			}).Debug("Current replay line")

			match := timeRegex.FindStringSubmatch(line)

			if match == nil {
				log.WithFields(log.Fields{
					"Regex": replayFile.TimestampRegex,
					"line":  line,
				}).Warn("Timestamp regex doesn't match the current line in the replay file, skipping line")
				continue
			}

			log.WithFields(log.Fields{
				"whole timestamp match":  match,
				"timeRegex.SubexpName()": timeRegex.SubexpNames(),
				"first_submatch":         match[1],
			}).Debug("Current replay line matches")

			// Put the names for the capture groups in a new map[string]string
			result := make(map[string]string)
			for i, name := range timeRegex.SubexpNames() {
				if i != 0 {
					log.Debug("i = ", i, ", result[name]= ", result[name])
					result[name] = match[i]
				}
			}

			startTime := result["hour"] + ":" + result["minute"] + ":" + result["second"]
			log.WithFields(log.Fields{
				"startTime": startTime,
			}).Debug("New Start Time")

			// Replace the line with the $[time||stamp] token for replacement
			augmentedLine := timeRegex.ReplaceAllString(line, "$[time||stamp]")
			log.WithFields(log.Fields{
				"augmentedLine": augmentedLine,
			}).Debug("New augmented line")

			logLine := loggensender.LogLineProperties{Text: augmentedLine, IntervalSecs: replayFile.RepeatInterval, IntervalStdDev: 0, StartTime: startTime, TimestampFormat: replayFile.TimestampFormat, Headers: replayFile.Headers}

			logLines = append(logLines, logLine)

		}
	}

	// Set individual log lines to global configs / defaults if need be
	for i := 0; i < len(logLines); i++ {
		logLines[i].HTTPClient = &confData.HTTPClient
		logLines[i].FileHandler = confData.FileHandler

		if logLines[i].OutputType == "" {
			logLines[i].OutputType = confData.OutputType
		}
		if logLines[i].HTTPLoc == "" {
			logLines[i].HTTPLoc = confData.HTTPLoc
		}
		if logLines[i].SyslogType == "" {
			logLines[i].SyslogType = confData.SyslogType
		}
		if logLines[i].SyslogLoc == "" {
			logLines[i].SyslogLoc = confData.SyslogLoc
		}
		if logLines[i].StartTime == "" {
			logLines[i].StartTime = targetStartTime.Format("15:04:05")
		}

	}

	log.WithFields(log.Fields{
		"count": len(logLines),
	}).Info("Finished storing normalized log lines")
	return
}

// validateConfFile ONLY does sanity checks on the values inside the conf file.
// The data file validation is handled elsewhere
func validateConfFile(confData *GlobalConfStore) {

	// File must have some defined inputs
	if len(confData.DataFiles) == 0 && len(confData.ReplayFiles) == 0 {
		log.WithFields(log.Fields{
			"output_type":  confData.OutputType,
			"data_files":   confData.DataFiles,
			"replay_files": confData.ReplayFiles,
		}).Fatal("Configuration file had 0 input files")
	}

	// Confirm the OutputType is valid
	if confData.OutputType != "http" && confData.OutputType != "syslog" && confData.OutputType != "file" {
		log.WithFields(log.Fields{
			"OutputType": confData.OutputType,
		}).Fatal("Output type in global conf is not in (http, syslog, file)")
	}

	// Confirm the HTTP Location is valid
	if r, _ := regexp.Compile("^https?://.*"); confData.OutputType == "http" && !(r.MatchString(confData.HTTPLoc)) {
		log.WithFields(log.Fields{
			"HTTPLocation": confData.HTTPLoc,
		}).Fatal("HTTP Location does not start with http:// or https://")
	}

	//Confirm SyslogLocation is valid
	if r, _ := regexp.Compile("^.*?:\\d+$"); confData.OutputType == "syslog" && !(r.MatchString(confData.SyslogLoc)) {
		log.WithFields(log.Fields{
			"SyslogLocation": confData.SyslogLoc,
		}).Fatal("Syslog Location is badly formatted according to regex: ^.*?:\\d+$")
	}

	// Confirm the SyslogType is valid
	if confData.OutputType == "syslog" && confData.SyslogType != "tcp" && confData.SyslogType != "udp" {
		log.WithFields(log.Fields{
			"SyslogType": confData.SyslogType,
		}).Fatal("Syslog type in global conf is not in (tcp, udp)")
	}

	// Confirm File Location is valid
	if confData.OutputType == "file" && confData.FileOutputPath == "" {
		log.Fatal("The output file path must be present, and non-blank in the global config if using the file output method")
	}

	// Loop over all the data files, if any are present
	if len(confData.DataFiles) > 0 {
		for i := 0; i < len(confData.DataFiles); i++ {
			// Confirm the path is non-blank
			if confData.DataFiles[i].Path == "" {
				log.Fatal("All data files must have a non-blank path in the global config")
			}
		}
	}

	// Loop over all replay files, if any are present
	if len(confData.ReplayFiles) > 0 {
		for _, replayFile := range confData.ReplayFiles {
			// Confirm the path is non-blank
			if replayFile.Path == "" {
				log.Fatal("All replay files must have a non-blank path in the global config")
			}

			// Confirm the timestamp regex compiles
			if _, err := regexp.Compile(replayFile.TimestampRegex); err != nil {
				log.WithFields(log.Fields{
					"regex":     replayFile.TimestampRegex,
					"error_msg": err,
				}).Fatal("This regex in the conf file is not valid in the Go regex parser")
			}

			// Confirm the timestamp format is not blank (I couldn't find an err returning time function)
			if replayFile.TimestampFormat == "" {
				log.Fatal("All replay files must have a TimestampFormat")
			}

			// Confirm that a Repeat Interval is present
			// This should be handled by JSON Marshaling, so I just need to check for zero
			if replayFile.RepeatInterval == 0 {
				log.WithFields(log.Fields{
					"repeatInterval": replayFile.RepeatInterval,
				}).Fatal("The repeat interval must be a non-zero integer")
			}

			// Confirm all the Headers have the needed fields, if any exist
			if len(replayFile.Headers) > 0 {
				for k := 0; k < len(replayFile.Headers); k++ {
					if replayFile.Headers[k].Header == "" || replayFile.Headers[k].Value == "" {
						log.WithFields(log.Fields{
							"Header": replayFile.Headers[k].Header,
							"Value":  replayFile.Headers[k].Value,
						}).Fatal("Both the header and value need to be non-zero in the replay file definition")
					}
				}
			}
		}
	}
}

// validateDataFile will do a sanity check on all values in a data file,
// displaying useful errors and aborting if need be
func validateDataFile(dataJSON *LogGenDataFile) {

	// Loop through all line objects in the file
	if len(dataJSON.Lines) > 0 {
		for _, logLine := range dataJSON.Lines {

			//Confirm Text field exists
			if logLine.Text == "" {
				log.WithFields(log.Fields{
					"lineJSON": logLine,
				}).Error("Text field cannot be empty string or missing in data file JSON")
				continue
			}

			// Confirm IntervalSecs or IntervalSecsMillis are not zero
			if logLine.IntervalSecs == 0 && logLine.IntervalMillis == 0 {
				log.WithFields(log.Fields{
					"lineJSON": logLine,
				}).Error("IntervalSecs and IntervalMillis fields cannot both be 0 or missing in data file JSON")
				continue
			}

			// IntervalStdDev can be zero... so no sanity checks possible here

			//Confirm Timestamp format field exists
			if logLine.TimestampFormat == "" {
				log.WithFields(log.Fields{
					"lineJSON": logLine,
				}).Error("TimestampFormat field cannot be empty string or missing in data file JSON")
				continue
			}

			// No good way to check for this only when necessary
			/*// Confirm the Start Time is valid
			if r, _ := regexp.Compile(`^\d\d:\d\d:\d\d`); !(r.MatchString(logLine.StartTime)) {
				log.WithFields(log.Fields{
					"StartTime": logLine.StartTime,
				}).Fatal("Start time must be of the form HH:mm:ss")
			}*/

			// Confirm all the Headers have the needed fields, if any exist
			if len(logLine.Headers) > 0 {
				for k := 0; k < len(logLine.Headers); k++ {
					if logLine.Headers[k].Header == "" || logLine.Headers[k].Value == "" {
						log.WithFields(log.Fields{
							"lineJSON": logLine,
						}).Fatal("Both the header and value need to be non-zero in the data file JSON")
					}
				}
			}

		}
	}

}

func main() {
	fmt.Println("Starting main program")

	// Read in the config file
	confText, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error_msg": err,
		}).Fatal("Something went amiss trying to read the global conf file")
	}

	// Unmarshal the Global Config JSON into a struct
	var confData GlobalConfStore
	err = json.Unmarshal(confText, &confData)
	if err != nil {
		log.WithFields(log.Fields{
			"error_msg": err,
		}).Fatal("Something went amiss on parsing the global config file. Has the config been validated as valid JSON?")
	}

	fmt.Println("Config File Parsed")

	validateConfFile(&confData)

	// Initialize the FileHandler if needed
	if confData.OutputType == "file" {
		f, err := os.Create(confData.FileOutputPath)
		if err != nil {
			log.WithFields(log.Fields{
				"FileOutputPath": confData.FileOutputPath,
				"error_msg":      err,
			}).Fatal("Error in creating the output file, exiting")
		}
		confData.FileHandler = f
		defer f.Close()
	}

	runQueue := make(chan loggensender.LogLineProperties)

	//Spawn worker pool to keep the queue processing
	for w := 1; w < workers; w++ {
		go loggensender.RunLogLine(runQueue)
	}

	// Add in some delay before starting because we're not sure how long it will take to parse the lines
	targetStartTime := time.Now().Add(5 * time.Second).Truncate(time.Second)

	// Create an object to store LogLines
	logLines := parseAndStoreLogLines(confData, targetStartTime)

	// Kick off sending of all log lines over a channel
	queueLogLines(logLines, targetStartTime, runQueue)

	fmt.Println("==== Successfully started the loggen process ====")

	// Set up a channel that will never receive data to keep main loop open
	delay := make(chan int)
	<-delay

}
