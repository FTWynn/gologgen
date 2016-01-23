![lologgen logo](http://i.imgur.com/xv2D2lE.png)

![Go report card](http://goreportcard.com/badge/ftwynn/gologgen) ![Travis Build Badge](https://travis-ci.org/FTWynn/gologgen.svg?branch=master)

gologgen is a generic log generator written in go. I'm writing it because I want to learn golang, and there was a need for a universal, well documented fake log generator at my job. All feedback is greatly appreciated.

## Installation

You can use the builds in any directory on the appropriate platform (setting the linux and osx builds to executable before you do). In addition to the binary, a config directory **must** be next to it with the config file format defined below.

## Configuration

The first needed file is gologgen.conf, which stores global variables in a JSON object. A working example is in the repo's config directory with all of the parameters filled out, but here are the current options:

Conf Parameter | Notes
--------- | -----
OutputType | "http" or "syslog"
httpLoc | URL of the http endpoint to send logs. Supports https.
SyslogLoc | Location to send syslog traffic, in the form of IP:port
SyslogType | "tcp" or "udp"
DataFiles | Array of objects describing DataFiles. Only contains "Path".
ReplayFiles | Array of objects describing ReplayFiles. Contains values described below.

As you might guess, the second thing you'll need is a combination of data files (JSON descriptions of log lines) and replay files (plain captures of raw log data). The parameters for each are below. Remember that these parameters go in the DataFile if going that route, or in the gologgen.conf if you're using Replayfiles. The idea is that you can drop in a capture as a replay, without going in and editing it (though you could add wildcards to it if you wanted). If that sounds confusing, just use the examples as a guide.

DataFile Parameter | Notes
--------- | -----
Text | Log message to write. The text is always interpreted literally except for the three wildcard segments: $[time,stamp], $[integer,integer], and $[string,string,...]. The first inserts the timestamp at the location. The second inserts a random integer between the two values. The third picks a random string and inserts it.
IntervalSecs | Interval in seconds to repeat the message. The minimum value is 1.
IntervalStdDev | Standard Deviation of the Interval if you want to add some randomness. Specified as a float.
TimestampFormat | The timestamp format to write on the message. See note below.
Headers | An array of objects with a Header and Value key, that correspond to http request headers

Replay Parameter | Notes
--------- | -----
Path | Path to the file. This is relative to the executable, and a good idea would be to use the config directory.
TimestampRegex | A go regular expression that pulls out the timestamp from the log line into named capture groups: year, month, day, hour, minute, second. Most of these are ignored (year, month, day), but the others are important.
TimestampFormat | The timestamp format to write on the message. See note below.
RepeatInterval | The number of seconds between replays of the file. Be mindful that if you set this to less than the timespan of your data file, things will eventually blow up. (I should probably fix that at some point...)

Most of the above is pretty self explanatory. The only exception being the TimestampFormat. Go does this odd thing when specifying timestamp formats, where you can express the date string however you like, but it **must** correspond to the date and time of:

    Mon Jan 2 15:04:05 MST 2006

... or as a numbers inclined person might see it...

    01/02 03:04:05PM '06 -0700

In the future there may be an enhancement to use YYYY-MM-DD syntax, but at the moment we're going with the raw golang behavior. Again, check the examples if it doesn't seem clear.

## Behavior Implications

There are a few implications to this structure.

1. You generally can't specify a log to be created more than once per second in a data file. You could list it as two separate items, but because the lowest increment worked on is 1 second, there's no single config for it.
2. There's currently no way to specify something like "Run every 5 seconds for a 10 minutes window, then stop for an hour." Your best bet presently is to use replay files, which support all the wildcards but not the std deviations, and toy with the replay interval compared to the data listed.

Log Level (DEBUG, INFO, WARN, ERROR) for the *gologgen application* is controlled by a runtime flag:

    gologgen_win_amd64 -level=INFO

This currently all goes to stdout.
