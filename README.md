# gologgen

gologgen is a flexible log generator written in go. I mostly wrote it because I wanted to learn golang, though I also wanted a different layout from the internal log generator we had.

## Setup

There are two files to be used when setting up gologgen. First, a gologgen.conf, and second a gologgen.data.

### gologgen.conf

Presently, this only has the http location, though there will be support for more later

    http_loc http://somewhere.thats.listening.com/a-place

### gologgen.data

This is a file that will contain the data to be randomized and generated.

TODO: Fill out what and how :-P
