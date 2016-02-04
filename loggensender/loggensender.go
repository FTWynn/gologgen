package loggensender

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ftwynn/gologgen/loggenmunger"

	log "github.com/Sirupsen/logrus"
)

// LogLineProperties holds all the data relevant to running a Log Line
type LogLineProperties struct {
	OutputType      string
	SyslogType      string
	SyslogLoc       string
	HTTPLoc         string              `json:"HTTPLoc"`
	Text            string              `json:"Text"`
	IntervalSecs    int                 `json:"IntervalSecs"`
	IntervalStdDev  float64             `json:"IntervalStdDev"`
	TimestampFormat string              `json:"TimestampFormat"`
	Headers         []LogLineHTTPHeader `json:"Headers"`
	StartTime       string              `json:"StartTime"`
	HTTPClient      *http.Client
	FileHandler     *os.File
}

// LogLineHTTPHeader holds the key and vlue for each header
type LogLineHTTPHeader struct {
	Header string `json:"Header"`
	Value  string `json:"Value"`
}

// DispatchLogs takes a slice of Log Lines and a time and fires the ones listed, re-adding them to the Run Table where the next run should go
func DispatchLogs(RunTable *map[time.Time][]LogLineProperties, ThisTime time.Time) {

	RunTableObj := *RunTable

	// If no log lines, clean up and exit
	if len(RunTableObj[ThisTime]) == 0 {
		delete(RunTableObj, ThisTime)
		log.WithFields(log.Fields{
			"time": ThisTime,
		}).Info("No logs to dispatch, exiting")
		return
	}

	log.WithFields(log.Fields{
		"time":  ThisTime,
		"count": len(RunTableObj[ThisTime]),
	}).Info("Starting Dispatch Logs")

	// get a rand object for later
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	lines := RunTableObj[ThisTime]
	for _, line := range lines {
		go RunLogLine(line, ThisTime)

		// Insert into RunTable for the next run
		// Randomize the Interval by specifying the std dev and adding the desired mean
		milliseconds := line.IntervalSecs * 1000
		stdDevMilli := line.IntervalStdDev * 1000.0
		nextInterval := int(r.NormFloat64()*stdDevMilli + float64(milliseconds))
		if nextInterval < 1000 {
			nextInterval = 1000
		}
		nextTime := ThisTime.Add(time.Duration(nextInterval) * time.Millisecond).Truncate(time.Second)
		log.WithFields(log.Fields{
			"line":     line.Text,
			"nextTime": nextTime,
		}).Info("SCHEDULED - Next log run")
		RunTableObj[nextTime] = append(RunTableObj[nextTime], line)

	}

	delete(RunTableObj, ThisTime)
	log.WithFields(log.Fields{
		"time": ThisTime,
	}).Info("Finished dispatching logs")
}

// RunLogLine runs an instance of a log line through the appropriate output
func RunLogLine(params LogLineProperties, sendTime time.Time) {

	// Randomize the text if need be
	var stringBody = []byte(loggenmunger.RandomizeString(params.Text, params.TimestampFormat))

	switch params.OutputType {
	case "http":
		go sendLogLineHTTP(params.HTTPClient, stringBody, params)
	case "syslog":
		go sendLogLineSyslog(stringBody, params)
	case "file":
		go sendLogLineFile(stringBody, params)
	}
}

// sendLogLineHTTP sends the log line to the http endpoint, retrying if need be
func sendLogLineHTTP(client *http.Client, stringBody []byte, params LogLineProperties) {
	// Post to HTTP
	log.WithFields(log.Fields{
		"line": string(stringBody),
	}).Info("Sending log over HTTP")

	req, err := http.NewRequest("POST", params.HTTPLoc, bytes.NewBuffer(stringBody))
	for _, header := range params.Headers {
		req.Header.Add(header.Header, header.Value)
	}
	log.WithFields(log.Fields{
		"request": req,
	}).Debug("Request object to send to Sumo")

	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"error_msg": err,
			"line":      string(stringBody),
		}).Error("Something went wrong with the http client")
		return
	}

	// For non 200 StatusCode, retry 5 times and then give up
	if resp.StatusCode != 200 {
		log.Debug("Non 200 response, retrying")
		for i := 0; i < 5; i++ {
			log.WithFields(log.Fields{
				"attemptNumber": i + 1,
			}).Debug("Retrying HTTP Post")
			resp2, err := client.Do(req)
			defer resp.Body.Close()
			if resp2.StatusCode == 200 && err == nil {
				break
			}
			if i == 4 {
				log.WithFields(log.Fields{
					"error_msg": err,
					"line":      string(stringBody),
				}).Error("Got non-200 response from HTTP Location and retries failed")
			}
			time.Sleep(time.Duration(10) * time.Second)
		}
	}
	log.WithFields(log.Fields{
		"statusCode": resp.StatusCode,
	}).Debug("Response from Sumo")
}

//sendLogLineSyslog sends the log on tcp/udp, WITHOUT retrying
func sendLogLineSyslog(stringBody []byte, params LogLineProperties) {
	log.WithFields(log.Fields{
		"line":     string(stringBody),
		"location": params.SyslogLoc,
	}).Info("Sending log to syslog")

	conn, err := net.Dial(params.SyslogType, params.SyslogLoc)
	if err != nil {
		log.WithFields(log.Fields{
			"error_msg":      err,
			"type":           params.SyslogType,
			"syslogLocation": params.SyslogLoc,
		}).Error("Failed to create syslog connection, abandoning")
	}
	defer conn.Close()

	fmt.Fprintf(conn, string(stringBody))
}

//sendLogLineFile writes log lines to a file
func sendLogLineFile(stringBody []byte, params LogLineProperties) {
	log.WithFields(log.Fields{
		"line": string(stringBody),
	}).Info("Writing log to file")

	_, err := params.FileHandler.Write(append(stringBody, []byte("\n")...))
	if err != nil {
		log.WithFields(log.Fields{
			"error_msg": err,
		}).Fatal("Error writing to file")
	}

}
