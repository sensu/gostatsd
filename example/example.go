package main

import (
	"log"

	"github.com/sensu/gostatsd/statsd"
)

func main() {
	f := func(m statsd.Metric) {
		log.Printf("%s", m)
	}
	r := statsd.MetricReceiver{":8125", statsd.HandlerFunc(f)}
	r.ListenAndReceive()
}
