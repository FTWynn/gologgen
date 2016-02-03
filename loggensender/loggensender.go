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
	"github.com/ftwynn/gologgen/loghelper"

	log15 "gopkg.in/inconshreveable/log15.v2"
)

var log log15.Logger

// LogLineProperties holds all the data relevant to running a Log Line
type LogLineProperties struct {
	OutputType      string
	SyslogType      string
	SyslogLoc       string
	HTTPLoc         string              `json:"HTTPLoc"`
	PostBody        string              `json:"Text"`
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

// I'm not really sure why this bit is required (and doesn't overwrite what's in main)... I may need to build my own logging library so I can grasp all the particulars
func init() {
	log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StdoutHandler))
	log = log15.New("function", log15.Lazy{Fn: loghelper.Log15LazyFunctionName})
}

// DispatchLogs takes a slice of Log Lines and a time and fires the ones listed, re-adding them to the Run Table where the next run should go
func DispatchLogs(RunTable *map[time.Time][]LogLineProperties, ThisTime time.Time) {

	log.Info("Starting Dispatch Logs")
	RunTableObj := *RunTable
	log.Info("Starting Dispatch Logs", "time", ThisTime, "length", len(RunTableObj[ThisTime]))

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
		log.Info("SCHEDULED - Next log run", "line", line.PostBody, "nextTime", nextTime)
		RunTableObj[nextTime] = append(RunTableObj[nextTime], line)

	}

	delete(RunTableObj, ThisTime)
	log.Info("Finished dispatching logs", "time", ThisTime)
}

// RunLogLine runs an instance of a log line through the appropriate output
func RunLogLine(params LogLineProperties, sendTime time.Time) {
	log.Info("Starting Individual Log Runner", "time", sendTime, "logline", params.PostBody)

	// Randomize the post body if need be
	var stringBody = []byte(loggenmunger.RandomizeString(params.PostBody, params.TimestampFormat))

	switch params.OutputType {
	case "http":
		go sendLogLineHTTP(params.HTTPClient, stringBody, params)
	case "syslog":
		go sendLogLineSyslog(stringBody, params)
	case "file":
		go sendLogLineFile(stringBody, params)
	}
	log.Info("Finished Individual Log Runner", "time", sendTime, "logline", params.PostBody)
}

// sendLogLineHTTP sends the log line to the http endpoint, retrying if need be
func sendLogLineHTTP(client *http.Client, stringBody []byte, params LogLineProperties) {
	// Post to Sumo
	log.Info("Sending log to Sumo over HTTP", "line", string(stringBody))
	req, err := http.NewRequest("POST", params.HTTPLoc, bytes.NewBuffer(stringBody))
	for _, header := range params.Headers {
		req.Header.Add(header.Header, header.Value)
	}
	log.Debug("Request object to send to Sumo", "request", req)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Error("Something went amiss on submitting to Sumo", "error_msg", err, "line", string(stringBody))
		return
	}
	if resp.StatusCode != 200 {
		log.Debug("Non 200 response, retrying")
		for i := 0; i < 5; i++ {
			log.Debug("Retrying", "attemptNumber", i+1)
			resp2, err := client.Do(req)
			defer resp.Body.Close()
			if resp2.StatusCode == 200 && err == nil {
				break
			}
			time.Sleep(time.Duration(10) * time.Second)
		}
	}
	log.Debug("Response from Sumo", "statusCode", resp.StatusCode)
}

//sendLogLineSyslog sends the log on tcp/udp, WITHOUT retrying
func sendLogLineSyslog(stringBody []byte, params LogLineProperties) {
	log.Info("Sending log to syslog", "line", string(stringBody), "location", params.SyslogLoc)

	conn, err := net.Dial(params.SyslogType, params.SyslogLoc)
	if err != nil {
		log.Error("Failed to create syslog connection, abandoning", "error_msg", err, "type", params.SyslogType, "syslogLocation", params.SyslogLoc)
	}
	defer conn.Close()

	fmt.Fprintf(conn, string(stringBody))
}

//sendLogLineFile writes log lines to a file
func sendLogLineFile(stringBody []byte, params LogLineProperties) {
	log.Info("Writing log to file", "line", string(stringBody))

	_, err := params.FileHandler.Write(append(stringBody, []byte("\n")...))
	if err != nil {
		log.Error("Error writing to file", "error_msg", err)
		panic(err)
	}

}
