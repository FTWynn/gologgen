package main

import (
	"encoding/json"
	"gologgen/loggenrunner"
	"io/ioutil"
	"time"

	log "github.com/Sirupsen/logrus"
)

// GlobalConfStore holds all the config data from the conf file
type GlobalConfStore struct {
	HTTPLoc     string               `json:"httpLoc"`
	OutputType  string               `json:"OutputType"`
	SyslogType  string               `json:"SyslogType"`
	SyslogLoc   string               `json:"SyslogLoc"`
	DataFiles   []DataFileMetaData   `json:"DataFiles"`
	ReplayFiles []ReplayFileMetaData `json:"ReplayFiles"`
}

// DataFileMetaData stores the configs around data files
type DataFileMetaData struct {
	Path string `json:"Path"`
}

// ReplayFileMetaData stores all the info around a replay file
type ReplayFileMetaData struct {
	Path           string `json:"Path"`
	TimestampRegex string `json:"TimestampRegex"`
}

//
func storeDataFileLogLines(confData GlobalConfStore) (dataJSON loggenrunner.LogGenDataFile) {
	// Read in the data files
	for i := 0; i < len(confData.DataFiles); i++ {
		dataText, err := ioutil.ReadFile(confData.DataFiles[i].Path)
		if err != nil {
			log.Error("Something went amiss on data file read: ", err)
		}

		// convert the data to somethign we can work with
		err = json.Unmarshal(dataText, &dataJSON)
		if err != nil {
			log.Error("something went amiss on parsing the data file: ", err)
			return
		}
	}

	// Set individual log lines to global configs if need be
	for i := 0; i < len(dataJSON.Lines); i++ {
		if dataJSON.Lines[i].OutputType == "" {
			dataJSON.Lines[i].OutputType = confData.OutputType
		}
		if dataJSON.Lines[i].HTTPLoc == "" {
			dataJSON.Lines[i].HTTPLoc = confData.HTTPLoc
		}
		if dataJSON.Lines[i].SyslogType == "" {
			dataJSON.Lines[i].SyslogType = confData.SyslogType
		}
		if dataJSON.Lines[i].SyslogLoc == "" {
			dataJSON.Lines[i].SyslogLoc = confData.SyslogLoc
		}

	}
	return dataJSON
}

func init() {
	log.SetLevel(log.InfoLevel)
}

func main() {
	// Read in the config file
	confText, err := ioutil.ReadFile("config/gologgen.conf")
	if err != nil {
		log.Error("something went amiss on conf file read")
		return
	}
	log.Debug("Read in conf from file: ", string(confText))

	// Unmarshal the Global Config JSON into a struct
	var confData GlobalConfStore
	err = json.Unmarshal(confText, &confData)
	if err != nil {
		log.Error("something went amiss on parsing the global config file")
		return
	}
	log.Debug("Parsed conf results", confData)

	// Create an object to stare DataFile LogLines
	dataJSON := storeDataFileLogLines(confData)

	RunTable := make(map[time.Time][]loggenrunner.LogLineProperties)

	// Add in some delay before starting off the ticker because we're not sure how long it will take to initialize our lines into the RunTable
	targetTickerTime := time.Now().Add(10 * time.Second).Truncate(time.Second)

	loggenrunner.InitializeRunTable(&RunTable, dataJSON.Lines, targetTickerTime)
	log.Debug("Finished RunTable:\n", RunTable)

	// Set up a Ticker and call the dispatcher to create the log lines
	tickerChannel := time.Tick(1 * time.Second)
	for thisTime := range tickerChannel {
		log.Debug("Tick for time: ", thisTime.Truncate(time.Second))
		go loggenrunner.DispatchLogs(&RunTable, thisTime.Truncate(time.Second))
	}

}
