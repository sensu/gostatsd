gostatsd
========

An implementation of [Etsy's][etsy] [statsd][statsd] in Go. This project was
forked from the original, github.com/kisielk/gostatsd, as we found that the
implementation choices and rapid release cycle of Atlassian's fork were not
a good fit for us.

Going forward, we will provide this project as a module, and attempt to make
minimal changes from the original implementation.

The project provides both a server called "gostatsd" which works much like
Etsy's version, but also provides a library for developing customized servers.

Building the server
-------------------
From the `gostatsd/` directory run `go build`. The binary will be built in place
and called `gostatsd`

Running the server
------------------
`gostatsd -help` gives a complete description of available options and their
defaults.

Sending metrics
---------------
The server listens for UDP packets on the address given by the `-l` flag,
aggregates them, then sends them to graphite server address given by the `-g`
flag.

The format of each metric is:

    <bucket name>:<value>|<type>\n

* `<bucket name>` is a string like `abc.def.g`, just like a graphite bucket name
* `<value>` is a string representation of a floating point number
* `<type>` is one of `c`, `g`, or `ms` for "counter", "gauge", and "timer"
respectively.

A single packet can contain multiple metrics, each ending with a newline.

A simple way to test your installation or send metrics from a script is to use
`echo` and the [netcat][netcat] utility `nc`:

    echo 'abc.def.g:10|c' | nc -w1 -u localhost 8125

Monitoring
----------
Currently you can get some basic idea of the status of the server by visiting the
address given by the `-c` option with your web browser.

Using the library
-----------------
In your source code:

    import "github.com/sensu/gostatsd/statsd"

Documentation can be found via `go doc github.com/sensu/gostatsd/statsd` or at
http://godoc.org/github.com/sensu/gostatsd/statsd

[etsy]: http://www.etsy.com
[statsd]: http://www.github.com/etsy/statsd
[netcat]: http://netcat.sourceforge.net/
