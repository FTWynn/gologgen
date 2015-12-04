package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
)

/* TODO: Fix these structs so the JSON derefs into them
type confStore struct {
	httpLoc string `json:"httpLoc"`
}

type dataStore struct {
	text string `json:"text"`
}*/

func main() {
	// Read in the config file
	confText, err := ioutil.ReadFile("gologgen.conf")
	if err != nil {
		log.Print("ERROR: something went amiss on conf file read")
		return
	}
	log.Print("DEBUG: Read in conf from file: ", string(confText))

	// Unmarshal the JSON into a map
	var cd map[string]string
	//var cd confStore //TODO: Fix the importing into a struct
	err2 := json.Unmarshal(confText, &cd)
	if err2 != nil {
		log.Print("ERROR: something went amiss on parse")
		return
	}
	log.Print("DEBUG: Parsed conf results", cd)
	log.Print("DEBUG: cd Type: ", reflect.TypeOf(cd))

	// Read in the data file
	dataText, err4 := ioutil.ReadFile("gologgen.data")
	if err4 != nil {
		log.Print("ERROR: something went amiss on data file read")
		return
	}
	log.Print("DEBUG: Read in data from file: ", string(dataText))

	// Convert the data into something we can work with
	// TODO: Probably should turn into struct as well
	var dataJSON map[string][]map[string]string
	err3 := json.Unmarshal(dataText, &dataJSON)
	if err3 != nil {
		log.Print("ERROR: something went amiss on parse")
		return
	}
	log.Print("DEBUG: Parse in data in memory: ", string(dataText))

	lines := dataJSON["lines"]
	log.Print("DEBUG: Lines parsed", lines)

	// Loop through lines and post to Sumo
	for _, v := range lines {
		var tester = []byte(v["text"])
		resp, err5 := http.Post(cd["httpLoc"], "text/plain", bytes.NewBuffer(tester))
		if err5 != nil {
			log.Print("ERROR: something went amiss on submitting to Sumo")
			return
		}
		defer resp.Body.Close()
		log.Print("Response from Sumo: ", resp)
	}

	/*// Test post, please ignore
	var tester = []byte("Test post, please ignore")
	resp, err := http.Post(cd["httpLoc"], "text/plain", bytes.NewBuffer(tester))
	defer resp.Body.Close()
	log.Print("Response from Sumo: ", resp)*/
}
