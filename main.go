package main

import (
	"encoding/json"
	"fmt"
	"gologgen/loggenrunner"
	"io/ioutil"
	"reflect"

	log "github.com/Sirupsen/logrus"
)

// ConfStore holds all the config data from the conf file
type ConfStore struct {
	HTTPLoc string `json:"httpLoc"`
}

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	// Read in the config file
	confText, err := ioutil.ReadFile("gologgen.conf")
	if err != nil {
		log.Error("something went amiss on conf file read")
		return
	}
	log.Debug("Read in conf from file: ", string(confText))

	// Unmarshal the JSON into a struct
	var cd ConfStore
	err = json.Unmarshal(confText, &cd)
	if err != nil {
		log.Error("something went amiss on parse")
		return
	}
	log.Debug("Parsed conf results", cd)
	log.Debug("cd Type: ", reflect.TypeOf(cd))

	// Read in the data file
	dataText, err4 := ioutil.ReadFile("gologgen.data")
	if err4 != nil {
		log.Error("something went amiss on data file read")
		return
	}
	log.Debug("Read in data from file: ", string(dataText))

	// Convert the data into something we can work with
	dataJSON := loggenrunner.LogGenDataFile{}
	err = json.Unmarshal(dataText, &dataJSON)
	if err != nil {
		log.Error("something went amiss on parse: ", err)
		return
	}
	log.Debug("Parse in data in memory: ", string(dataText))
	log.Debug("Resulted parsed data: ", dataJSON)
	log.Debug("Type of parsed data: ", reflect.TypeOf(dataJSON.Lines[0]))
	log.Debug("Resulted parsed data: ", dataJSON.Lines[2])

	lines := dataJSON.Lines

	// Loop through lines and post to Sumo
	for _, line := range lines {
		if line.HTTPLoc == "" {
			line.HTTPLoc = cd.HTTPLoc
		}
		go loggenrunner.RunLogLine(line)
	}

	// This will kill all the goroutines when enter is typed in the console
	var input string
	fmt.Scanln(&input)
}
