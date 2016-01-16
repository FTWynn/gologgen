# History and Thoughts

### 2015-12-13

Not everything seems appropriate for a README, and given that I'm learning this as I go, I figure I should write my thoughts and observations as they come to me in _something_ that might help other people walk through things.

My original list of objectives to work through was this simple:
* GoLogGen Roadmap
  * v1
    * Posts to http endpoint
  * v2
    * Reads from a file and posts
  * v3
    * Reads from file and repeatedly posts at uniform intervals
  * v4
    * Add randomness to the posting
  * v5
    * Add file values for randomness
  * v6
    * Add bounded values to random gen in things like names and ips
  * v7
    * Add headers in file and post
  * v8
    * Address timestamp generation
  * v9
    * Add support for replays from simple captures

The iterations were quick, and everything seemed to go pretty smoothly... until that last line.

That's not quite true. Figuring out how to smoothly convert between byte strings and builtin strings was really tough, and figuring out which supported a "Reader" when wasn't clear. The time library is mostly great, but the formatting is just plain weird with using a constant date instead of "YYYY-MM-DD" style syntax. I'm trying to avoid using third party libraries for this first project, the only exception thus far being the logging library which allows me to set the level much more easily.

Anyway, back to the replay roadblock... the log line data design that emerged around as I went was built around an interval and a standard deviation. This is great for independent log lines (which I'd implemented with goroutines), but as soon as you introduce dependance (logouts shouldn't precede logins, for example), then things get much more complicated.

Never mind that I also needed to extract timestamps in a general way, which either meant a bank of regexes (which I don't mind, but seems brittle), or including that in the replay file. AND I needed to figure out how to set a start time if it was important to start the replay at the top of the hour.

In other words, the independent, let every log line be a goroutine and take care of itself approach didn't seem like it was going to work anymore. I struggled with coming up with other models for a bit, but then I stumbled on the idea of a ticker in the time library. It's just a channel that jots down the time every interval you specify. Perhaps, rather than leaving a spinning goroutine for each log line, I should let the main loop fire a dispatcher and figure out which log lines should be generated each interval. I can start with a short duration and see how it scales, or batch everything by the timeframe instead in case 1 second isn't long enough for bigger configs.

At least... that'll be the first big refactor I undertake here.

### 2015-12-13

Boy... not really understanding pointers has come back to bite me today, but I think I've gotten the hang of it now.

I had an inspiration at lunch for a way to only loop over the Log Lines necessary for a given Second (where I'm starting the Ticker Interval). Basically, store a two dimensional array, with time as an index and a list of LogLineProperty objects, and get that needed slice whenever the Ticker comes around. I can then delete that piece from there.

This should take care of a few timing problems I ran into previously, and maybe even the dependency problem if I get clever with it.

But yeah... passing pointers around is one I haven't really messed around with before.

### 2015-12-15

While I've continued to make progress on features, like writing to Syslog and importing from a replay file, I've run into a bug I can't track down.

It seems if I try to fire too many events in one second, they disappear. I can't really explain it. I tried some more printf logging, but it just isn't enough to give me a clue as to why the lines would disappear. Maybe some sort of lag? Maybe the delete line gets a little ambitious and kills lines that still need running? Maybe there's just too much to do in one second?

A look around for debuggers shows positively dreadful results in the go arena. Nothing seems to elegantly handle concurrency (which I suppose makes sense in a "pause the world" way). In either case, this one is particularly frustrating.

... ugh, finally. There was a negative mod that caused the timestamps to be shifted backwards from the target time instead of forwards. Bleh.

### 2016-01-12

I put down this project for a while, but picked it up again on having to create a build for a coworker. It strikes me that the documentation wasn't nearly clear enough, and setting log levels would be super important from a binary perspective.

Onward to it!

### 2016-01-16

I moved the builds to the releases page in github. Seems obvious on reflection, but that stuff never makes sense at the moment.

I've also realized that I'm going to need to reorganize the project functions. "Runner" isn't very descriptive, and I should probably figure out an easier way to break them apart so I can make tests better.

Finally, there's a design decision that's weighing on me as I consider adding support for Slack output. It's essentially an HTTP Post, so nothing too complicated there, but I need to figure out if I'm assuming all the logs go to one output, or if each input file/line can be configured to a different output. That'll require a pretty significant re-org... but at this point it seems like I'm in for one of those either way.

Just thought I'd share the problems of a post 1.0 release.

=====

I was having some trouble deciding what broken out packages should be responsible for. I envision four main areas of the program:

1. Read in config data
2. Parse and store data for runtime
3. Augment data at runtime
4. Send data

It's pretty clear the last should be its own package. There should be a package for data munging, which is definitely step 3, and main should probably handle step 1. Step 2 however is a little ambiguous. I would think main programs usually handle run loops, which make it a good candidate for that, but as I add features and adjust configs, it almost seems like a data munging job.

I went with main covering steps 1 and 2 for now, though I suppose it could always be changed later.

=====

I also find myself unsure of where my data types should go. they don't seem limited enough to go in the helper packages, though putting them in the big package seems a little weird as well.

Eh, we'll start with main for now.

... is what I thought... up until I couldn't figure out how to import structs from main and use them in sub packages.

I guess the pattern would then be, put the structs in the most specific sub package that will use them, and import them everywhere else.
