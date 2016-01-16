package loggensender

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/ftwynn/gologgen/loggenmunger"
	"github.com/ftwynn/gologgen/loghelper"

	log15 "gopkg.in/inconshreveable/log15.v2"
)

var log log15.Logger

// LogLineProperties holds all the data relevant to running a Log Line
type LogLineProperties struct {
	OutputType      string
	HTTPLoc         string `json:"HTTPLoc"`
	SyslogType      string
	SyslogLoc       string
	PostBody        string  `json:"Text"`
	IntervalSecs    int     `json:"IntervalSecs"`
	IntervalStdDev  float64 `json:"IntervalStdDev"`
	TimestampFormat string  `json:"TimestampFormat"`
	SumoCategory    string  `json:"SumoCategory"`
	SumoHost        string  `json:"SumoHost"`
	SumoName        string  `json:"SumoName"`
	StartTime       string  `json:"StartTime"`
	HTTPClient      *http.Client
}

// I'm not really sure why this bit is required... I may need to build my own logging library so I can grasp all the particulars
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

// RunLogLine makes runs an instance of a log line through the appropriate output
func RunLogLine(params LogLineProperties, sendTime time.Time) {
	log.Info("Starting Individual Log Runner", "time", sendTime, "logline", params.PostBody)

	// Randomize the post body if need be
	var stringBody = []byte(loggenmunger.RandomizeString(params.PostBody, params.TimestampFormat))

	switch params.OutputType {
	case "http":
		go sendLogLineHTTP(params.HTTPClient, stringBody, params)
	case "syslog":
		go sendLogLineSyslog(stringBody, params)
	}
	log.Info("Finished Individual Log Runner", "time", sendTime, "logline", params.PostBody)
}

// sendLogLineHTTP sends the log line to the http endpoint, retrying if need be
func sendLogLineHTTP(client *http.Client, stringBody []byte, params LogLineProperties) {
	// Post to Sumo
	log.Info("Sending log to Sumo over HTTP", "line", string(stringBody))
	req, err := http.NewRequest("POST", params.HTTPLoc, bytes.NewBuffer(stringBody))
	req.Header.Add("X-Sumo-Category", params.SumoCategory)
	req.Header.Add("X-Sumo-Host", params.SumoHost)
	req.Header.Add("X-Sumo-Name", params.SumoName)
	log.Debug("Request object to send to Sumo", "request", req)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Error("Something went amiss on submitting to Sumo", "error", err, "line", string(stringBody))
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
		log.Error("Failed to create syslog connection, abandoning", "error", err, "type", params.SyslogType, "syslogLocation", params.SyslogLoc)
	}
	defer conn.Close()

	fmt.Fprintf(conn, string(stringBody))
}
