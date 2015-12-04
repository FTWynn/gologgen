package loggenrunner

import (
	"bytes"
	"math/rand"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

// DataStore holds all the data info for a given simulated log line
type DataStore struct {
	Text string `json:"text"`
}

// RunLogLine makes repeated calls to an endpoint given the configs of the log line
func RunLogLine(httpLoc string, postBody string, interval int) {
	log.Info("Starting log runner")

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Store the post body... will probably need to radnomize later
	var tester = []byte(postBody)

	// Begin loop to post the value until we're done
	for {
		// Post to Sumo
		resp, err := http.Post(httpLoc, "text/plain", bytes.NewBuffer(tester))
		if err != nil {
			log.Error("something went amiss on submitting to Sumo")
			return
		}
		defer resp.Body.Close()
		log.Debug("Response from Sumo: ", resp)

		// Sleep until the next run
		// Rnadomize the sleep by specifying the std dev and adding the desired mean... targeting 3%
		milliseconds := interval * 1000
		nextInterval := int(r.NormFloat64()*2000.0 + float64(milliseconds))
		time.Sleep(time.Duration(nextInterval) * time.Millisecond)
	}

}
