package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"

	log "github.com/Sirupsen/logrus"
)

// ConfStore holds all the config data from the conf file
type ConfStore struct {
	HTTPLoc string `json:"httpLoc"`
}

// DataStore holds all the data info for a given simulated log line
type DataStore struct {
	Text string `json:"text"`
}

func init() {
	// Only log the debug severity or above.
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

	// Unmarshal the JSON into a map
	//var cd map[string]string
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
		var tester = []byte(v["text"])
		resp, err5 := http.Post(cd.HTTPLoc, "text/plain", bytes.NewBuffer(tester))
		if err5 != nil {
			log.Error("something went amiss on submitting to Sumo")
			return
		}
		defer resp.Body.Close()
		log.Debug("Response from Sumo: ", resp)
	}
}
