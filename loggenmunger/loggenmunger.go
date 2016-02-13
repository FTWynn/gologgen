package loggenmunger

import (
	"errors"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

// RandomizeString takes a string, looks for the random tokens
// (int, string, and timestamp), and replaces them
func RandomizeString(text string, timeformat string) string {
	log.WithFields(log.Fields{
		"text":       text,
		"timeformat": timeformat,
	}).Debug("Starting String Randomization")

	// Bail if we can't get any randomizers
	goodstring, err := regexp.MatchString(`\$\[[^\]]+\]`, text)
	if err != nil {
		log.WithFields(log.Fields{
			"error_msg": err,
			"text":      text,
			"regex":     `\$\[[^\]]+\]`,
		}).Error("Something broke on parsing the text string with a regular expression")
	}

	//Return original string if 0 randomizers
	if !goodstring {
		log.Debug("Found no random tokens, returning the original string")
		return text
	}

	// Find all randomizing tokens
	re := regexp.MustCompile(`\$\[[^\]]+\]`)
	randos := re.FindAllString(text, -1)
	log.WithFields(log.Fields{
		"num":          len(randos),
		"randomTokens": randos,
	}).Debug("Found random tokens")

	// Create a list of new strings to be inserted where the tokens were
	var newstrings []string

	// Append the properly randomized values to the newstrings slice
	for _, rando := range randos {
		value, err := getOneToken(rando, timeformat)
		if err != nil {
			log.WithFields(log.Fields{
				"error":        err,
				"randomTokens": rando,
			}).Warn("Couldn't get value from token list. Using default TOKEN_ERROR")
			value = "TOKEN_ERROR"
		}

		newstrings = append(newstrings, value)
	}

	// We can use the same regex that found the tokens to split up the original string
	// That gives us clear insertion points for our selected tokens
	nonRandomStrings := re.Split(text, -1)
	var newLogLine []string

	for i := 0; i < len(nonRandomStrings); i++ {
		newLogLine = append(newLogLine, nonRandomStrings[i])
		if i != len(nonRandomStrings)-1 {
			newLogLine = append(newLogLine, newstrings[i])
		}
	}

	log.Debug("Randomization complete: ", "newString - ", strings.Join(newLogLine, ""))

	return strings.Join(newLogLine, "")
}

func getOneToken(tokenString string, timeformat string) (string, error) {
	replacer := strings.NewReplacer("$[", "", "]", "")

	// Take off the leading and trailing formatting
	tempstring := replacer.Replace(tokenString)
	log.WithFields(log.Fields{
		"tempstring": tempstring,
	}).Debug("Removing the formatting from the item list")

	// Split the randomizer into individual items
	itemList := strings.Split(tempstring, "||")
	log.WithFields(log.Fields{
		"itemList": itemList,
		"count":    len(itemList),
	}).Debug("Splitting the random tokens up")

	// Numeric ranges will only have two items for an upper and lower bound,
	// timestamps have "time" and "stamp", all the rest are string groups
	var randType string
	num0, err := strconv.Atoi(string(itemList[0]))
	num1, err2 := strconv.Atoi(string(itemList[1]))
	log.WithFields(log.Fields{
		"num":       num0,
		"error_msg": err,
	}).Debug("Parsing entry 0 as a number")
	log.WithFields(log.Fields{
		"num":       num1,
		"error_msg": err2,
	}).Debug("Parsing entry 1 as a number")

	// Set the logic for categorization here, so it's not buried in the switch
	switch {
	case len(itemList) == 2 && err == nil && err2 == nil:
		randType = "Number"
	case itemList[0] == "time" && itemList[1] == "stamp":
		randType = "Timestamp"
	default:
		randType = "Category"
	}

	log.WithFields(log.Fields{
		"type": randType,
	}).Debug("Finished determining token type")

	switch randType {
	case "Category":
		return itemList[rand.Intn(len(itemList))], nil
	case "Number":
		// Get a random number in the range
		diff := num1 - num0
		log.Debug("Difference from second and first numbers: ", "diff - ", diff)
		tempnum := rand.Intn(diff)
		log.Debug("Random number from zero adjusted spread: ", "rand - ", tempnum)
		log.Debug("Random number adjusted to range and string converted: ", "rand - ", strconv.Itoa(tempnum+num0))
		return strconv.Itoa(tempnum + num0), nil
	case "Timestamp":
		t := time.Now()
		timeformatted, err := formatTimestamp(t, timeformat)
		if err != nil {
			log.WithFields(log.Fields{
				"error_msg": err,
			}).Error("Formatting the timestamp broke")
		}
		return timeformatted, nil
	}

	// Failure case. Should never happen.
	return "", errors.New("Unknown error trying to get a token from the list: " + tokenString)
}

func formatTimestamp(t time.Time, timeformat string) (string, error) {
	log.Debug("Current time: ", "now - ", t)
	var timeformatted string
	switch timeformat {
	case "epoch":
		timeformatted = strconv.FormatInt(t.Unix(), 10)
	case "epochmilli":
		timeformatted = strconv.FormatInt(t.UnixNano()/1000000, 10)
	case "epochnano":
		timeformatted = strconv.FormatInt(t.UnixNano(), 10)
	default:
		timeformatted = t.Format(timeformat)
	}
	log.Debug("Formatted time: ", "now - ", timeformatted)
	return timeformatted, nil
}
