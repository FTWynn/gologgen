package loggenmunger

import (
	"testing"
	"time"
)

func TestGetOneToken(t *testing.T) {
	postitiveCases := []struct {
		tokenString, timeFormat string
	}{
		{"$[0||5]", "Jan 02 15:04:05"},
		{"$[Post||Thing||Stuff]", "Jan 02 15:04:05"},
	}
	for _, c := range postitiveCases {
		output, err := getOneToken(c.tokenString, c.timeFormat)
		if err != nil || output == "" {
			t.Errorf("Failed positive case: {%q,%q} >> %q - %q", c.tokenString, c.timeFormat, output, err)
		}
	}

	negativeCases := []struct {
		tokenString, timeFormat string
	}{
		{"$[time||stamp]", "bogus"},
		{"$[time||stamp]", "Feb 01 12:02:02"},
	}
	for _, c := range negativeCases {
		output, err := getOneToken(c.tokenString, c.timeFormat)
		if err != nil && output != "TIME_FORMAT_ERROR" {
			t.Errorf("Failed negative case: {%q,%q} >> %q - %q", c.tokenString, c.timeFormat, output, err)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	type FormatTimestampCases struct {
		t             time.Time
		timeformat    string
		desiredOutput string
	}
	referenceTime1, _ := time.Parse(time.RFC3339, "2016-01-01T00:00:00+00:00")
	referenceTime2, _ := time.Parse(time.RFC3339, "2016-02-12T05:42:21+00:00")
	referenceTime3, _ := time.Parse(time.RFC3339, "2015-12-29T22:08:41+00:00")
	cases := []FormatTimestampCases{
		// Good Formatting Cases
		{referenceTime1, "epoch", "1451606400"},
		{referenceTime1, "epochmilli", "1451606400000"},
		{referenceTime1, "epochnano", "1451606400000000000"},
		{referenceTime1, "Jan 02 15:04:05", "Jan 01 00:00:00"},
		{referenceTime1, "2006-01-02 15:04:05", "2016-01-01 00:00:00"},
		{referenceTime2, "Jan 02 15:04:05", "Feb 12 05:42:21"},
		{referenceTime3, "Jan 02 15:04:05", "Dec 29 22:08:41"},
		// Bad Formatting Cases
		{referenceTime1, "2008-01-02 17:02:01", "TIME_FORMAT_ERROR"},
		{referenceTime1, "2004", "TIME_FORMAT_ERROR"},
		{referenceTime1, "", "TIME_FORMAT_ERROR"},
		{referenceTime2, "bogus", "TIME_FORMAT_ERROR"},
	}
	for _, c := range cases {
		output, err := formatTimestamp(c.t, c.timeformat)
		if output != c.desiredOutput {
			t.Errorf("Failed case: {%q,%q,%q} >> %q - %q", c.t, c.timeformat, c.desiredOutput, output, err)
		}
	}
}
