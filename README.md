# gologgen

gologgen is a generic log generator written in go. I mostly wrote it because I wanted to learn golang, and there were a few needs in my job for a good fake log generator. As such, all feedback is welcome, because anything I learn now will hopefully suck less in the future.

## Setup

There are two files to be used when setting up gologgen. First, a gologgen.conf, and second a gologgen.data.

### gologgen.conf

This stores global configs, and potentially defaults. All in JSON.

    {
      "httpLoc" : "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV0o8ZcEo-edSG28OScCSzOzHtojKTRId_fimMMYzbIBk1f7ciR2FE6JHXKONkhlHohT30cD1ZeCrDvvQAhMbgjjjRxEQBcn-M3sh9PRMVtt6A==",
      "OutputType" : "http"
    }


### gologgen.data

This file contains individual loglines to be generated, along with each line's parameters. Timestamps, random numbers, and random selections of words can be specified within the strings.

    {
      "lines" : [
        {
          "Text" : "$[time,stamp] Test Random Numbers: $[0,5] $[8,10]",
          "IntervalSecs" : 2,
          "IntervalStdDev" : 0.5,
          "TimestampFormat" : "2006-01-02 15:04:05",
          "SumoCategory" : "OverwrittenCategory1",
          "SumoName" : "OverwrittenName1",
          "SumoHost" : "OverwrittenHost1"
        },
        {
          "Text" : "$[time,stamp] Test Random Category: $[Post,Thing,Stuff]",
          "IntervalSecs" : 4,
          "IntervalStdDev" : 3,
          "TimestampFormat" : "2006-01-02 15:04:05",
          "SumoCategory" : "OverwrittenCategory2",
          "SumoName" : "OverwrittenName2",
          "SumoHost" : "OverwrittenHost2"
        },
        {
          "Text" : "Test No Randoms: Ta da!",
          "IntervalSecs" : 1,
          "IntervalStdDev" : 0.2,
          "TimestampFormat" : "2006-01-02 15:04:05",
          "SumoCategory" : "OverwrittenCategory3",
          "SumoName" : "OverwrittenName4",
          "SumoHost" : "OverwrittenHost5"
        },
        {
          "Text" : "$[time,stamp] Rock Solid Repeater",
          "IntervalSecs" : 5,
          "IntervalStdDev" : 0,
          "StartTime" : "12:03:01",
          "TimestampFormat" : "2006-01-02 15:04:05",
          "SumoCategory" : "OverwrittenCategory3",
          "SumoName" : "OverwrittenName6",
          "SumoHost" : "OverwrittenHost6"
        }
        ]
    }

## Roadmap

There are a number of things I'd like to get to, but I'm not good enough to know how feasible any of them are yet. In rough order of what I'll attempt next, here's what's currently on the list.

* Add syslog output
* Add support for replays from simple captures
* Make parameters optional in the JSONData
* Validate that the conf and data files are correctly formatted
* Provide builds
* Randomizer should check for valid syntax inside the group
* Some way to link either log lines together (logouts should wait on logins, and logins shouldn't repeat without logouts)
* More efficiently pass around the http and syslog clients
