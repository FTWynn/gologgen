package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	// Read in the config file
	confText, err := ioutil.ReadFile("gologgen.conf")
	if err != nil {
		return
	}

	// Unmarshal the JSON into a map
	var cd map[string]string
	err2 := json.Unmarshal(confText, &cd)
	if err2 != nil {
		return
	}

	// Test post, please ignore
	var tester = []byte("Test post, please ignore")
	//c := bufio.Reader.ReadString(tester)
	//copy(c, tester)
	resp, err := http.Post(cd["httpLoc"], "text/plain", bytes.NewBuffer(tester))
	defer resp.Body.Close()
	fmt.Println(resp)
}
