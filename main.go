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
	err2 := json.Unmarshal(confText, &cd)
	if err2 != nil {
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
	// TODO: Probably should turn into struct as well
	var dataJSON map[string][]map[string]string
	//var dataJSON []DataStore
	err3 := json.Unmarshal(dataText, &dataJSON)
	if err3 != nil {
		log.Error("something went amiss on parse")
		return
	}
	log.Debug("Parse in data in memory: ", string(dataText))

	lines := dataJSON["lines"]
	log.Debug("Lines parsed", lines)

	// Loop through lines and post to Sumo
	for _, v := range lines {
		var tester = v["text"]
		go loggenrunner.RunLogLine(cd.HTTPLoc, tester, 1)
	}

	// This will kill al the goroutines when enter is typed in the console
	var input string
	fmt.Scanln(&input)
}
