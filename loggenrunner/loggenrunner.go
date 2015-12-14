package loggenrunner

import (
	"bytes"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

// LogGenDataFile represents a data file
type LogGenDataFile struct {
	Lines []LogLineProperties `json:"lines"`
}

// LogLineProperties holds all the data relevant to running a Log Line
type LogLineProperties struct {
	HTTPLoc         string  `json:"HTTPLoc"`
	PostBody        string  `json:"Text"`
	IntervalSecs    int     `json:"IntervalSecs"`
	IntervalStdDev  float64 `json:"IntervalStdDev"`
	TimestampFormat string  `json:"TimestampFormat"`
	SumoCategory    string  `json:"SumoCategory"`
	SumoHost        string  `json:"SumoHost"`
	SumoName        string  `json:"SumoName"`
	StartTime       string  `json:"StartTime"`
}

// randomizeString takes a string, looks for the random tokens (int, string, and timestamp), and replaces them
func randomizeString(text string, timeformat string) string {
	// Bail if we can't get any randomizers
	goodstring, err := regexp.MatchString(`\$\[[^\]]+\]`, text)
	if err != nil {
		log.Error("Something broke on parsing the text string with a regular expression")
	}

	//Return original string if no randomizers
	if !goodstring {
		log.Debug("Found no random tokens: ", text)
		return text
	}

	// Find all randomizing tokens
	re := regexp.MustCompile(`\$\[[^\]]+\]`)
	randos := re.FindAllString(text, -1)
	log.Debug("Random tokens: ", randos)

	// Create a list of new strings to be inserted where the tokens were
	var newstrings []string
	replacer := strings.NewReplacer("$[", "", "]", "")

	// Append the properly randomized values to the newstrings slice
	for _, rando := range randos {
		// Take off the leading and trailing formatting
		tempstring := replacer.Replace(rando)
		log.Debug("tempstring: ", tempstring)

		// Split the rnadomizer into individual items
		tempstrings := strings.Split(tempstring, ",")
		log.Debug("tempstrings: ", tempstrings)

		// Numeric ranges will only have two items for an upper and lower bound, timestamps have "time" and "stamp", all the rest are string groups
		var randType string
		num0, err := strconv.Atoi(string(tempstrings[0]))
		num1, err2 := strconv.Atoi(string(tempstrings[1]))
		log.Debug("num0 parsed: ", num0, err)
		log.Debug("num1 parsed: ", num1, err2)
		log.Debug("Length of tempstrings: ", len(tempstrings))
		log.Debug("Numbers?: ", len(tempstrings) == 2 && err == nil && err2 == nil)
		log.Debug("Timestamp?: ", tempstrings[0] == "time" && tempstrings[1] == "stamp")

		switch {
		case len(tempstrings) == 2 && err == nil && err2 == nil:
			randType = "Number"
		case tempstrings[0] == "time" && tempstrings[1] == "stamp":
			randType = "Timestamp"
		default:
			randType = "Category"
		}

		switch randType {
		case "Category":
			log.Debug("Treating as Category")
			newstrings = append(newstrings, tempstrings[rand.Intn(len(tempstrings))])
		case "Number":
			log.Debug("Treating as Number")

			// Get a random number in the range
			diff := num1 - num0
			log.Debug("diff parsed: ", diff)
			tempnum := rand.Intn(diff)
			log.Debug("random number: ", tempnum)
			log.Debug("random number as string: ", strconv.Itoa(tempnum+num0))
			newstrings = append(newstrings, strconv.Itoa(tempnum+num0))
		case "Timestamp":
			t := time.Now()
			log.Debug("Current time: ", t)
			timeformatted := t.Format(timeformat)
			log.Debug("Formatted time: ", timeformatted)

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

	log.Info("Randomization complete. New string: ", strings.Join(newLogLine, ""))

	return strings.Join(newLogLine, "")
}

// sendLogLineHTTP sends the log line to the http endpoint, retrying if need be
func sendLogLineHTTP(client *http.Client, stringBody []byte, params LogLineProperties) {
	// Post to Sumo
	log.Info("Sending log to Sumo: ", string(stringBody))
	req, err := http.NewRequest("POST", params.HTTPLoc, bytes.NewBuffer(stringBody))
	req.Header.Add("X-Sumo-Category", params.SumoCategory)
	req.Header.Add("X-Sumo-Host", params.SumoHost)
	req.Header.Add("X-Sumo-Name", params.SumoName)
	log.Debug("Request object to send to Sumo: ", req)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Error("Something went amiss on submitting to Sumo: ", err)
		return
	}
	if resp.StatusCode != 200 {
		log.Debug("Non 200 response, retrying")
		for i := 0; i < 5; i++ {
			log.Debug("Retry #", i+1)
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

// InitializeRunTable will take a slice of LogLines and start times and put the various lines in their starting slots in the map
func InitializeRunTable(RunTable *map[time.Time][]LogLineProperties, Lines []LogLineProperties, tickerStart time.Time) {
	RunTableObj := *RunTable
	for _, line := range Lines {
		// Get the log line target start time
		var targetTime time.Time
		if line.StartTime == "" {
			targetTime = tickerStart
		} else {
			re := regexp.MustCompile(`\d+`)
			targetHourMinSec := re.FindAllString(line.StartTime, -1)
			targetHour, _ := strconv.Atoi(targetHourMinSec[0])
			targetMin, _ := strconv.Atoi(targetHourMinSec[1])
			targetSec, _ := strconv.Atoi(targetHourMinSec[2])
			loc, _ := time.LoadLocation("America/Los_Angeles")
			targetTime = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), targetHour, targetMin, targetSec, 0, loc)
		}
		log.Debug("The target time is translated to: ", targetTime)
		log.Debug("The start time of the ticker is: ", tickerStart)
		diff := targetTime.Sub(tickerStart)
		log.Debug(" The diff between them is: ", diff)
		diffMod := int(diff.Seconds()) % line.IntervalSecs

		switch {
		case targetTime.Equal(tickerStart) || targetTime.After(tickerStart):
			log.Debug("Target is equal to or after start, so appending to Run Table as is")
			RunTableObj[targetTime] = append(RunTableObj[targetTime], line)
		case targetTime.Before(tickerStart):
			if diffMod == 0 {
				log.Debug("TickerStart is a multiple of Target's interval, so setting to TickerStart")
				RunTableObj[tickerStart] = append(RunTableObj[tickerStart], line)
			} else {
				log.Debug("Setting a start after ticker start to: ", tickerStart.Add(time.Duration(diffMod)*time.Second))
				RunTableObj[tickerStart.Add(time.Duration(diffMod)*time.Second)] = append(RunTableObj[tickerStart.Add(time.Duration(diffMod)*time.Second)], line)
			}
		}

	}
}

// RunLogLine makes runs an instance of a log line through the appropriate channel
func RunLogLine(params LogLineProperties, sendTime time.Time) {
	log.Info("Starting log runner for logline: ", params.PostBody)

	client := &http.Client{}

	// Randomize the post body if need be
	var stringBody = []byte(randomizeString(params.PostBody, params.TimestampFormat))

	go sendLogLineHTTP(client, stringBody, params)

}

// DispatchLogs takes a slice of Log Lines and a time and fires the ones listed, re-adding them to the Run Table where the next run should go
func DispatchLogs(RunTable *map[time.Time][]LogLineProperties, ThisTime time.Time) {
	log.Debug("Starting Dispatch Logs")
	RunTableObj := *RunTable

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
		nextTime := ThisTime.Add(time.Duration(nextInterval) * time.Millisecond).Truncate(time.Second)
		log.Debug("Next log run for \"", line.PostBody, "\" set for ", nextTime)
		RunTableObj[nextTime] = append(RunTableObj[nextTime], line)

	}

	delete(RunTableObj, ThisTime)
}
