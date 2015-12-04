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

//RunLogLineParams holds all the data to be passed to RunLogLine
type RunLogLineParams struct {
	HTTPLoc        string
	PostBody       string
	IntervalSecs   int
	IntervalStdDev float64
}

// RunLogLine makes repeated calls to an endpoint given the configs of the log line
func RunLogLine(HTTPLoc string, PostBody string, IntervalSecs int, IntervalStdDev float64) {
	log.Info("Starting log runner")

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Store the post body... will probably need to randomize later
	var tester = []byte(PostBody)

	// Begin loop to post the value until we're done
	for {
		// Post to Sumo
		resp, err := http.Post(HTTPLoc, "text/plain", bytes.NewBuffer(tester))
		if err != nil {
			log.Error("something went amiss on submitting to Sumo")
			return
		}
		defer resp.Body.Close()
		log.Debug("Response from Sumo: ", resp)

		// Sleep until the next run
		// Randomize the sleep by specifying the std dev and adding the desired mean... targeting 3%
		milliseconds := IntervalSecs * 1000
		stdDevMilli := IntervalStdDev * 1000.0
		nextInterval := int(r.NormFloat64()*stdDevMilli + float64(milliseconds))
		time.Sleep(time.Duration(nextInterval) * time.Millisecond)
	}

}
