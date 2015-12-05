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

// RunLogLineParams holds all the data to be passed to RunLogLine
type RunLogLineParams struct {
	HTTPLoc        string
	PostBody       string
	IntervalSecs   int
	IntervalStdDev float64
}

// randomizeString takes a string, looks for the random tokens (int and string), and replaces them, returning a []byte
func randomizeString(text string) []byte {
	goodstring, err := regexp.MatchString(`\$\[[^\]]+\]`, text)
	if err != nil {
		log.Error("Something broke on parsing the text string with a regular expression")
	}

	//Return original string if no randomizers
	if !goodstring {
		return []byte(text)
	}

	re := regexp.MustCompile(`\$\[[^\]]+\]`)
	randos := re.FindAllString(text, -1)
	log.Debug("Random tokens: ", randos)

	var newstrings []string
	replacer := strings.NewReplacer("$[", "", "]", "")

	// Append the properly randomized values to the newstrings slice
	for _, rando := range randos {
		// Take off the leading and trailing formatting
		tempstring := replacer.Replace(rando)
		log.Debug("tempstring: ", tempstring)

		// Split into individual items
		tempstrings := strings.Split(tempstring, ",")
		log.Debug("tempstrings: ", tempstrings)

		// Numeric ranges will only have two items for an upper and lower bound, all the rest are string groups
		// TODO: check for both elements, not just the first... maybe with a switch?
		var randType string
		num0, err := strconv.Atoi(string(tempstrings[0]))
		num1, err2 := strconv.Atoi(string(tempstrings[1]))
		log.Debug("num0 parsed: ", num0, err)
		log.Debug("num1 parsed: ", num1, err2)
		log.Debug("Length of tempstrings: ", len(tempstrings))
		log.Debug("Category switch: ", len(tempstrings) == 2 && err == nil && err2 == nil)

		if len(tempstrings) == 2 && err == nil && err2 == nil {
			randType = "Number"
		} else {
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

	return []byte(strings.Join(newLogLine, ""))
}

// RunLogLine makes repeated calls to an endpoint given the configs of the log line
func RunLogLine(HTTPLoc string, PostBody string, IntervalSecs int, IntervalStdDev float64) {
	log.Info("Starting log runner")

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Begin loop to post the value until we're done
	for {
		// Randomize the post body if need be
		var stringBody = randomizeString(PostBody)

		// Post to Sumo
		log.Info("Sending log to Sumo: ", stringBody)
		resp, err := http.Post(HTTPLoc, "text/plain", bytes.NewBuffer(stringBody))
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
