# gologgen

gologgen is a generic log generator written in go. I'm writing it because I want to learn golang, and there was a need for a good fake log generator at my job. As such, please let me know if I've done anything poorly, because anything I learn now will help me suck less in the future.

I'm also leaving the entire commit history here (at least for now), so that others can point and laugh at my mistakes... and maybe newbies will see how badly you have to start at something before you (eventually?) get better.

## Setup

The first needed file is gologgen.conf, which stores global variables. A working example is in the config directory. Second, you will need either a gologgen.data file, or a replay file. The format for each of these is listed in the example gologgen.conf file.

Most of it's pretty self explanatory. The only exception being the timestamp format. Go does this odd thing when specifying timestamp formats, where you can express the date string however you like, but it **must** correspond to the date:

    Mon Jan 2 15:04:05 MST 2006

... or as a numbers inclined person might see it...

    01/02 03:04:05PM '06 -0700

I may clean this up in the future with a YYYY-MM-DD shim, but for now, this is what we've got.

## Usage

There are a few implications to this setup.

1. You generally can't specify a log to be created more than once per second in a data file. You could list it as two separate items, but because the lowest increment worked on is 1 second, there's no single config for it.
2. There's currently no way to specify something like "Run every 5 seconds for a 10 minutes window, then stop for an hour." Your best bet presently is to use replay files, which support all the wildcards, but not the std deviations.

## Roadmap

There are a number of things I'd like to get to, but I'm not quite good enough to know how feasible any of them are yet. In rough order of what I'll attempt next, here's what's currently on the list.

* Add a "fast forward" capability to populate the logs as fast as possible, rather than in live time
* Provide builds
* ... tests???
* ... docs???
