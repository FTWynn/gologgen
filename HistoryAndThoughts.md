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
