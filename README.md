![lologgen logo](http://i.imgur.com/xv2D2lE.png)

![Travis Build Badge](https://travis-ci.org/FTWynn/gologgen.svg?branch=master) ![Go report card](http://goreportcard.com/badge/ftwynn/gologgen)

#### Current Release (Big Behavior Change): [3.0.0](https://github.com/FTWynn/gologgen/releases/tag/v3.0.0) - 2/15/2016

gologgen is a generic log generator written in go. I'm writing it because I want to learn golang, and there was a need for a universal, well documented fake log generator at my job. All feedback is greatly appreciated.

## Installation

Grab the latest release from the [Releases Page](https://github.com/FTWynn/gologgen/releases). You can use the builds in any directory on the appropriate platform (setting the linux and osx builds to executable before you do). In addition to the binary, you'll need a config file, and some number of data or replay files.

## Getting Started

There are a number of config examples provided in the config directory. You can start with a simple one, or shoot for the batteries included example. The config file ignores any values that aren't related to the OutputType.

You can either name your config gologgen.conf and put it in a directory named "config" next to the executable, or you can specify a path with the conf runtime flag.

    ./gologgen_linux_amd64 -conf=simple1.conf

You might want to tweak the log level as well if you're curious about the steps it's taking. WARN is the default.

     ./gologgen_linux_amd64 -conf=simple1.conf -level=DEBUG

The generated log lines either come from data files (JSON descriptions of log lines) or replay files (a capture of live log data). An example of each is in the repo.

## Global Configuration File

This file stores global variables in a JSON object. By default, the program will look for gologgen.conf in a config directory alongside the executable, but you can set the conf to what and wherever with the *conf* flag at runtime. The file only needs the configs related to the OutputType, and will safely ignore unrelated configs if they are present. The current options are:

Conf Parameter | Notes
--------- | -----
OutputType | "http", "syslog", or "file"
httpLoc | URL of the http endpoint to send logs. Supports https.
SyslogLoc | Location to send syslog traffic, in the form of IP:port
SyslogType | "tcp" or "udp"
FileOutputPath | Path of the file to write out to. Will *overwrite* whatever already exists.
DataFiles | Array of objects describing DataFiles. Only contains "Path".
ReplayFiles | Array of objects describing ReplayFiles. Contains values described below.

## Data File

A Data File is a JSON description of log lines. The parameters for each are below.

DataFile Parameter | Notes
--------- | -----
Text | Log message to write. The text is always interpreted literally except for the three wildcard formats. See below for details.
IntervalSecs | Interval in seconds to repeat the message. The minimum value is 1.
IntervalStdDev | Standard Deviation of the Interval if you want to add some randomness. Specified as a float.
IntervalMillis | Same as interval, but in Milliseconds. One of the two fields must be provided, and IntervalMillis takes precedence.
IntervalStdDevMillis | Standard Deviation of the Interval on a milliseconds scale. Provided as an Integer.
TimestampFormat | The timestamp format to write on the message. See note below.
StartTime | A string in the form of HH:mm:ss that denotes a start time to start the message sending. If the program begins earlier than this time, it will fire at the appropriate time. If the program starts after this time, then it will fire on the first multiple of the interval time after the program starts.
Headers | An array of objects with a Header and Value key, that correspond to http request headers

## Replay File

Replay files are log captures from other devices, which gologgen will then re-parse out and send. As such, the configuration for these actually goes in the *global conf file*. Gologgen will still look for replacement tokens in replay files, so if you want to add those in you can do that too.

Replay Parameter | Notes
--------- | -----
Path | Path to the file. This is relative to the executable.
TimestampRegex | A go regular expression that pulls out the timestamp from the log line into named capture groups: year, month, day, hour, minute, second. Be sure to match *the whole timestamp*, otherwise pieces of it won't get replaced on regeneration.
TimestampFormat | The timestamp format to write on the message. See note below.
RepeatInterval | The number of seconds between replays of the file. Be mindful that if you set this to less than the timespan of your data file, things will eventually blow up. (I should probably fix that at some point...)
Headers | An array of objects with a Header and Value key, that correspond to http request headers

## Wildcard Formats

gologgen provides support for a few wildcard types in the Data File Text line, as well as lines in Replay Files.

Timestamp Insertion:

    $[time||stamp]

Random Integer Generation:

    $[2||10]

Random String Selection:

    $[Thing1||Thing2||Thing3...||ThingN]

Timestamps will always be formatted according to the appropriate formatting regex in the config. Integers on the left must be smaller than integers on the right. String lists can be of any length, but they cannot be nested.

## A Note about Go Timestamp Formats

Most of the above is pretty self explanatory. The only exception being the TimestampFormat. Go does this odd thing when specifying timestamp formats, where you can express the date string however you like, but it **must** correspond to the date and time of:

    Mon Jan 2 15:04:05 MST 2006

... or as a numbers inclined person might see it...

    01/02 03:04:05PM '06 -0700

In the future there may be an enhancement to use YYYY-MM-DD syntax, but at the moment we're going with the raw golang behavior. Again, check the examples if it doesn't seem clear.

You can also specify one of three values for epoch time instead: epoch, epochmilli, and epochnano.

## Behavior Implications

There are a few implications to this structure.

1. There's currently no way to specify something like "Run every 5 seconds for a 10 minutes window, then stop for an hour." Your best bet presently is to use replay files, which support all the wildcards but not the std deviations, and toy with the replay interval compared to the data listed.

## Logging for gologgen

Log Level (DEBUG, INFO, WARN, ERROR) for the *gologgen application* is controlled by a runtime flag:

    gologgen_win_amd64 -level=INFO

This currently all goes to stdout.
