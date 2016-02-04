package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/ftwynn/gologgen/loggensender"

	log "github.com/Sirupsen/logrus"
)

var confPath string

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
	flag.StringVar(&level, "level", "WARN", "a string")
	flag.StringVar(&confPath, "conf", "config/gologgen.conf", "a string")

	flag.Parse()

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
		log.SetLevel(log.WarnLevel)
	}

}

// InitializeRunTable will take a slice of LogLines and start times and put the various lines in their starting slots in the map
func InitializeRunTable(RunTable *map[time.Time][]loggensender.LogLineProperties, Lines []loggensender.LogLineProperties, tickerStart time.Time) {
	RunTableObj := *RunTable
	for _, line := range Lines {
		log.Debug("========== New Line ==========")
		log.WithFields(log.Fields{
			"time": line.StartTime,
		}).Debug("The literal time string")

		// Get the log line target start time
		var targetTime time.Time
		if line.StartTime == "" {
			targetTime = tickerStart
		} else {
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
			log.Debug("Target is equal to or after start, so appending to Run Table as is")
			RunTableObj[targetTime] = append(RunTableObj[targetTime], line)
		case targetTime.Before(tickerStart):
			if diffMod == 0 {
				log.Debug("TickerStart is a multiple of Target's interval, so setting to TickerStart")
				RunTableObj[tickerStart] = append(RunTableObj[tickerStart], line)
			} else {
				log.WithFields(log.Fields{
					"startTime": tickerStart.Add(time.Duration(diffMod) * time.Second),
				}).Debug("Setting a start after ticker start")
				RunTableObj[tickerStart.Add(time.Duration(diffMod)*time.Second)] = append(RunTableObj[tickerStart.Add(time.Duration(diffMod)*time.Second)], line)
			}
		}

	}

	log.WithFields(log.Fields{
		"length": len(RunTableObj),
	}).Info("Total RunTable buildup")

}

// storeDataFileLogLines takes the conf data, gets the associated files, and puts them in a big list of LogLine Objects
func storeDataFileLogLines(confData GlobalConfStore) (logLines []loggensender.LogLineProperties) {
	log.WithFields(log.Fields{
		"confData":        confData,
		"logLines length": len(logLines),
	}).Info("Entering storeDataFileLogLines")

	// Return if no data files
	if len(confData.DataFiles) == 0 && len(confData.ReplayFiles) == 0 {
		log.Error("No data files or replay files were found in the global config file.")
	}

	dataJSON := LogGenDataFile{}

	// First, read in any data files
	for i := 0; i < len(confData.DataFiles); i++ {
		dataText, err := ioutil.ReadFile(confData.DataFiles[i].Path)
		if err != nil {
			log.WithFields(log.Fields{
				"error_msg": err,
				"file_path": confData.DataFiles[i].Path,
			}).Error("Couldn't read in the data file, so ignoring and moving on to the next data file")
			continue
		}

		// Convert the data to something we can work with
		err = json.Unmarshal(dataText, &dataJSON)
		if err != nil {
			log.WithFields(log.Fields{
				"error_msg": err,
				"file_path": confData.DataFiles[i].Path,
			}).Error("Couldn't read in text from found data file, so ignoring and moving on to the next data file")
			continue
		}

		// Add the parsed fields to the result value
		for i := 0; i < len(dataJSON.Lines); i++ {
			// Bail if no Text
			if dataJSON.Lines[i].Text == "" {
				log.WithFields(log.Fields{
					"json": dataJSON.Lines[i],
				}).Error("Line in data file must have a Text value")
				continue
			}
			logLines = append(logLines, dataJSON.Lines[i])
		}
	}

	// Second, read in the replay files
	for i := 0; i < len(confData.ReplayFiles); i++ {
		replayFile := confData.ReplayFiles[i]

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

			augmentedLine := timeRegex.ReplaceAllString(line, "$[time,stamp]")
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

	}

	log.WithFields(log.Fields{
		"count": len(logLines),
	}).Info("Finished storing normalized log lines")
	return
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
			"text":      confText,
		}).Fatal("Something went amiss on parsing the global config file: ")
	}

	fmt.Println("Config File Parsed")

	log.WithFields(log.Fields{
		"confData": confData,
	}).Debug("Parsed global config results")

	// Bail on essential missing configs
	if confData.OutputType == "" || (len(confData.DataFiles) == 0 && len(confData.ReplayFiles) == 0) {
		log.WithFields(log.Fields{
			"output_type":  confData.OutputType,
			"data_files":   confData.DataFiles,
			"replay_files": confData.ReplayFiles,
		}).Fatal("Configuration was either missing an output type, or had 0 input files")
	}

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

	// Create an object to store DataFile LogLines
	logLines := storeDataFileLogLines(confData)

	RunTable := make(map[time.Time][]loggensender.LogLineProperties)

	// Add in some delay before starting off the ticker because we're not sure how long it will take to initialize our lines into the RunTable
	targetTickerTime := time.Now().Add(5 * time.Second).Truncate(time.Second)

	InitializeRunTable(&RunTable, logLines, targetTickerTime)
	log.WithFields(log.Fields{
		"RunTable": RunTable,
	}).Debug("Finished RunTable")

	fmt.Println("LogLines imported")

	fmt.Println("==== Starting the main event loop (set the log level to INFO for more detail) ====")

	// Set up a Ticker and call the dispatcher to create the log lines
	tickerChannel := time.Tick(1 * time.Second)
	for thisTime := range tickerChannel {
		log.WithFields(log.Fields{
			"thisTime": thisTime.Truncate(time.Second),
		}).Debug("Tick for time")
		go loggensender.DispatchLogs(&RunTable, thisTime.Truncate(time.Second))
	}

}
