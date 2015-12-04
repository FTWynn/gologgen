package loggenrunner

import (
	"bytes"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// DataStore holds all the data info for a given simulated log line
type DataStore struct {
	Text string `json:"text"`
}

// RunLogLine makes repeated calls to an endpoint given the configs of the log line
func RunLogLine(httpLoc string, postBody string) {
	log.Info("Starting log runner")
	var tester = []byte("PostBody")
	resp, err5 := http.Post("HTTPLiteral", "text/plain", bytes.NewBuffer(tester))
	if err5 != nil {
		log.Error("something went amiss on submitting to Sumo")
		return
	}
	defer resp.Body.Close()
	log.Debug("Response from Sumo: ", resp)
}
