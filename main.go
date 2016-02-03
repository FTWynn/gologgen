package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/ftwynn/gologgen/loggensender"
	"github.com/ftwynn/gologgen/loghelper"

	log15 "gopkg.in/inconshreveable/log15.v2"
)

var log log15.Logger
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

	switch level {
	case "DEBUG":
		log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlDebug, log15.StdoutHandler))
	case "INFO":
		log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlInfo, log15.StdoutHandler))
	case "WARN":
		log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlWarn, log15.StdoutHandler))
	case "ERROR":
		log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StdoutHandler))
	default:
		log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlWarn, log15.StdoutHandler))
	}

	log = log15.New("function", log15.Lazy{Fn: loghelper.Log15LazyFunctionName})
}

// InitializeRunTable will take a slice of LogLines and start times and put the various lines in their starting slots in the map
func InitializeRunTable(RunTable *map[time.Time][]loggensender.LogLineProperties, Lines []loggensender.LogLineProperties, tickerStart time.Time) {
	RunTableObj := *RunTable
	for _, line := range Lines {
		log.Debug("========== New Line ==========")
		log.Debug("The literal time string", "time", line.StartTime)
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
		log.Debug("The target time is translated to", "targetTime", targetTime)
		log.Debug("The start time of the ticker is", "tickerStart", tickerStart)
		diff := targetTime.Sub(tickerStart)
		log.Debug("The diff between target and tickerStart is", "diff", diff)
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
				log.Debug("Setting a start after ticker start", "startTime", tickerStart.Add(time.Duration(diffMod)*time.Second))
				RunTableObj[tickerStart.Add(time.Duration(diffMod)*time.Second)] = append(RunTableObj[tickerStart.Add(time.Duration(diffMod)*time.Second)], line)
			}
		}

	}
	log.Info("Total RunTable buildup", "length", len(RunTableObj))

}

// storeDataFileLogLines takes the conf data, gets the associated files, and puts them in a big list of LogLine Objects
func storeDataFileLogLines(confData GlobalConfStore) (logLines []loggensender.LogLineProperties) {
	log.Info("Entering storeDataFileLogLines", "confData", confData, "logLines length", len(logLines))

	// Return if no data files
	if len(confData.DataFiles) == 0 && len(confData.ReplayFiles) == 0 {
		log.Error("No data files or replay files were found in the global config file.")
	}

	dataJSON := LogGenDataFile{}

	// First, read in any data files
	for i := 0; i < len(confData.DataFiles); i++ {
		dataText, err := ioutil.ReadFile(confData.DataFiles[i].Path)
		if err != nil {
			log.Error("Something went amiss on data file read: ", err)
		}

		// Convert the data to something we can work with
		err = json.Unmarshal(dataText, &dataJSON)
		if err != nil {
			log.Error("Something went amiss on parsing the data file: ", err)
			return
		}

		// Add the parsed fields to the result value
		for i := 0; i < len(dataJSON.Lines); i++ {
			// Bail if no Text
			if dataJSON.Lines[i].PostBody == "" {
				log.Error("Data Line must have a Text value", "json", dataJSON.Lines[i])
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
			log.Error("Something went amiss on replay file read: ", err)
		}
		defer file.Close()

		// Set the timestamp regex
		var timeRegex = regexp.MustCompile(replayFile.TimestampRegex)

		// Scan the file through line by line
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			log.Debug("Scanning file: ", file)
			line := scanner.Text()
			log.Debug("Current replay line: ", line)
			match := timeRegex.FindStringSubmatch(line)
			log.Debug("Current replay line matches: ", match)

			log.Debug("timeRegex.SubexpName(): ", timeRegex.SubexpNames())
			log.Debug("match: ", match[1])
			// Put the names for the capture groups in a new map[string]string
			result := make(map[string]string)
			for i, name := range timeRegex.SubexpNames() {
				if i != 0 {
					log.Debug("i = ", i, ", result[name]= ", result[name])
					result[name] = match[i]
				}
			}

			log.Debug("Remapping to named object: ", result)
			startTime := result["hour"] + ":" + result["minute"] + ":" + result["second"]
			log.Debug("New Start Time: ", startTime)
			augmentedLine := timeRegex.ReplaceAllString(line, "$[time,stamp]")
			log.Debug("New augmented line: ", augmentedLine)
			logLine := loggensender.LogLineProperties{PostBody: augmentedLine, IntervalSecs: replayFile.RepeatInterval, IntervalStdDev: 0, StartTime: startTime, TimestampFormat: replayFile.TimestampFormat, Headers: replayFile.Headers}
			log.Debug("New LogLine Object: ", logLine)

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

	log.Info("Finished storing normalized log lines", "count", len(logLines))
	return
}

func main() {
	log.Info("Starting main program")

	// Read in the config file
	confText, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Error("Something went amiss on global conf file read", "error_msg", err, "text", confText)
		return
	}

	// Unmarshal the Global Config JSON into a struct
	var confData GlobalConfStore
	err = json.Unmarshal(confText, &confData)
	if err != nil {
		log.Error("Something went amiss on parsing the global config file: ", "error_msg", err, "text", confText)
		return
	}
	log.Info("Parsed global config results", "results", confData)

	// Bail on essential missing configs
	if confData.OutputType == "" || (len(confData.DataFiles) == 0 && len(confData.ReplayFiles) == 0) {
		log.Error("Configuration was missing either an output type, or input files", "confg", confData)
		return
	}

	// Initialize the FileHandler if needed
	if confData.OutputType == "file" {
		f, err := os.Create(confData.FileOutputPath)
		if err != nil {
			log.Error("Error in creating the output file", "FileOutputPath", confData.FileOutputPath)
			return
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
	log.Debug("Finished RunTable:\n", "RunTable", RunTable)

	log.Info("==================== Starting the main event loop ==================")

	// Set up a Ticker and call the dispatcher to create the log lines
	tickerChannel := time.Tick(1 * time.Second)
	for thisTime := range tickerChannel {
		log.Debug("Tick for time", "thisTime", thisTime.Truncate(time.Second))
		go loggensender.DispatchLogs(&RunTable, thisTime.Truncate(time.Second))
	}

}
