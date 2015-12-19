package loggenrunner

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "gopkg.in/inconshreveable/log15.v2"
)

// LogGenDataFile represents a data file
type LogGenDataFile struct {
	Lines []LogLineProperties `json:"lines"`
}

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

// randomizeString takes a string, looks for the random tokens (int, string, and timestamp), and replaces them
func randomizeString(text string, timeformat string) string {
	// Bail if we can't get any randomizers
	goodstring, err := regexp.MatchString(`\$\[[^\]]+\]`, text)
	if err != nil {
		log.Error("Something broke on parsing the text string with a regular expression", "error_msg", err)
	}

	//Return original string if 0 randomizers
	if !goodstring {
		log.Debug("Found no random tokens: ", "text", text)
		return text
	}

	// Find all randomizing tokens
	re := regexp.MustCompile(`\$\[[^\]]+\]`)
	randos := re.FindAllString(text, -1)
	log.Debug("A found random tokens", "randomTokens", randos, "num", len(randos))

	// Create a list of new strings to be inserted where the tokens were
	var newstrings []string
	replacer := strings.NewReplacer("$[", "", "]", "")

	// Append the properly randomized values to the newstrings slice
	for _, rando := range randos {
		// Take off the leading and trailing formatting
		tempstring := replacer.Replace(rando)
		log.Debug("Removing the formatting from the items: ", "tempstring", tempstring)

		// Split the randomizer into individual items
		tempstrings := strings.Split(tempstring, ",")
		log.Debug("Splitting the random tokens up: ", "tempstrings", tempstrings, "count", len(tempstrings))

		// Numeric ranges will only have two items for an upper and lower bound, timestamps have "time" and "stamp", all the rest are string groups
		var randType string
		num0, err := strconv.Atoi(string(tempstrings[0]))
		num1, err2 := strconv.Atoi(string(tempstrings[1]))
		log.Debug("Parsing entry 0 as a number: ", "num", num0, "error", err)
		log.Debug("Parsing entry 1 as a number: ", "num", num1, "error", err2)

		switch {
		case len(tempstrings) == 2 && err == nil && err2 == nil:
			randType = "Number"
		case tempstrings[0] == "time" && tempstrings[1] == "stamp":
			randType = "Timestamp"
		default:
			randType = "Category"
		}

		log.Debug("What type of token is this?", "type", randType)

		switch randType {
		case "Category":
			newstrings = append(newstrings, tempstrings[rand.Intn(len(tempstrings))])
		case "Number":
			// Get a random number in the range
			diff := num1 - num0
			log.Debug("Difference from second and first numbers", "diff", diff)
			tempnum := rand.Intn(diff)
			log.Debug("Random number from zero adjusted spread", "rand", tempnum)
			log.Debug("Random number adjusted to range and string converted", "rand", strconv.Itoa(tempnum+num0))
			newstrings = append(newstrings, strconv.Itoa(tempnum+num0))
		case "Timestamp":
			t := time.Now()
			log.Debug("Current time", "now", t)
			timeformatted := t.Format(timeformat)
			log.Debug("Formatted time", "now", timeformatted)

			newstrings = append(newstrings, timeformatted)
		}
	}

	nonRandomStrings := re.Split(text, -1)
	var newLogLine []string

	for i := 0; i < len(nonRandomStrings); i++ {
		newLogLine = append(newLogLine, nonRandomStrings[i])
		if i != len(nonRandomStrings)-1 {
			newLogLine = append(newLogLine, newstrings[i])
		}
	}

	log.Info("Randomization complete", "newString", strings.Join(newLogLine, ""))

	return strings.Join(newLogLine, "")
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
		log.Error("Something went amiss on submitting to Sumo", "error", err)
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
	//log.Debug("Response from Sumo: ", resp)
}

//sendLogLineSyslog sends the log on tcp/udp, WITHOUT retrying
func sendLogLineSyslog(stringBody []byte, params LogLineProperties) {
	conn, err := net.Dial(params.SyslogType, params.SyslogLoc)
	if err != nil {
		log.Error("Failed to create syslog connection, abandoning", "error", err)
	}
	defer conn.Close()
	// Post to Syslog
	log.Info("Sending log to Syslog", "line", string(stringBody))
	fmt.Fprintf(conn, string(stringBody))
}

// RunLogLine makes runs an instance of a log line through the appropriate channel
func RunLogLine(params LogLineProperties, sendTime time.Time) {
	log.Info("Starting Individual Log Runner", "time", sendTime, "logline", params.PostBody)

	// Randomize the post body if need be
	var stringBody = []byte(randomizeString(params.PostBody, params.TimestampFormat))

	switch params.OutputType {
	case "http":
		go sendLogLineHTTP(params.HTTPClient, stringBody, params)
	case "syslog":
		go sendLogLineSyslog(stringBody, params)
	}
	log.Info("Finished Individual Log Runner", "time", sendTime, "logline", params.PostBody)
}

// DispatchLogs takes a slice of Log Lines and a time and fires the ones listed, re-adding them to the Run Table where the next run should go
func DispatchLogs(RunTable *map[time.Time][]LogLineProperties, ThisTime time.Time) {

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
}
