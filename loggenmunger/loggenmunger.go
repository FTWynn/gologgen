package loggenmunger

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ftwynn/gologgen/loghelper"

	log15 "gopkg.in/inconshreveable/log15.v2"
)

var log log15.Logger

func init() {
	log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlInfo, log15.StdoutHandler))
	log = log15.New("function", log15.Lazy{Fn: loghelper.Log15LazyFunctionName})
}

// RandomizeString takes a string, looks for the random tokens (int, string, and timestamp), and replaces them
func RandomizeString(text string, timeformat string) string {
	log.Debug("Starting String Randomization", "text", text, "timeFormat", timeformat)

	// Bail if we can't get any randomizers
	goodstring, err := regexp.MatchString(`\$\[[^\]]+\]`, text)
	if err != nil {
		log.Error("Something broke on parsing the text string with a regular expression", "error_msg", err, "text", text, "regex", `\$\[[^\]]+\]`)
	}

	//Return original string if 0 randomizers
	if !goodstring {
		log.Debug("Found no random tokens: ", "text", text)
		return text
	}

	// Find all randomizing tokens
	re := regexp.MustCompile(`\$\[[^\]]+\]`)
	randos := re.FindAllString(text, -1)
	log.Debug("A found random tokens", "num", len(randos), "randomTokens", randos)

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

	log.Debug("Randomization complete", "newString", strings.Join(newLogLine, ""))

	return strings.Join(newLogLine, "")
}
