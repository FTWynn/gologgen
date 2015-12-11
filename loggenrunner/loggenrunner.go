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

// DataStore holds all the data info for a given simulated log line
type DataStore struct {
	Text string `json:"text"`
}

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

// RunLogLine makes repeated calls to an endpoint given the configs of the log line
//func RunLogLine(HTTPLoc string, PostBody string, IntervalSecs int, IntervalStdDev float64, TimeFormat string, SumoCategory string, SumoHost string, SumoName string) {
func RunLogLine(params LogLineProperties) {
	log.Info("Starting log runner for logline: ", params.PostBody)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	client := &http.Client{}

	// Begin loop to post the value until we're done
	for {
		// Randomize the post body if need be
		var stringBody = []byte(randomizeString(params.PostBody, params.TimestampFormat))

		// Post to Sumo
		log.Info("Sending log to Sumo: ", string(stringBody))
		req, err := http.NewRequest("POST", params.HTTPLoc, bytes.NewBuffer(stringBody))
		req.Header.Add("X-Sumo-Category", params.SumoCategory)
		req.Header.Add("X-Sumo-Host", params.SumoHost)
		req.Header.Add("X-Sumo-Name", params.SumoName)
		resp, err := client.Do(req)
		if err != nil {
			log.Error("something went amiss on submitting to Sumo")
			return
		}
		defer resp.Body.Close()
		//log.Debug("Response from Sumo: ", resp)

		// Sleep until the next run
		// Randomize the sleep by specifying the std dev and adding the desired mean... targeting 3%
		milliseconds := params.IntervalSecs * 1000
		stdDevMilli := params.IntervalStdDev * 1000.0
		nextInterval := int(r.NormFloat64()*stdDevMilli + float64(milliseconds))
		time.Sleep(time.Duration(nextInterval) * time.Millisecond)
	}

}
