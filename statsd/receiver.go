package statsd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
)

// DefaultMetricsAddr is the default address on which a MetricReceiver will listen
const DefaultMetricsAddr = ":8125"

// Objects implementing the Handler interface can be used to handle metrics for a MetricReceiver
type Handler interface {
	HandleMetric(m Metric)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as metric handlers
type HandlerFunc func(Metric)

// HandleMetric calls f(m)
func (f HandlerFunc) HandleMetric(m Metric) {
	f(m)
}

// MetricReceiver receives data on its listening port and converts lines in to Metrics.
// For each Metric it calls r.Handler.HandleMetric()
type MetricReceiver struct {
	Addr    string  // UDP address on which to listen for metrics
	Handler Handler // handler to invoke
}

// ListenAndReceive is like ListenAndReceiveContext, but context.Background()
// is used. Use ListenAndReceive if you don't need to control cancellation.
func (r *MetricReceiver) ListenAndReceive() error {
	return r.ListenAndReceiveContext(context.Background())
}

// ListenAndReceiveContext listens on the UDP network address of r.Addr and then calls
// Receive to handle the incoming datagrams. If Addr is blank then DefaultMetricsAddr is used.
func (r *MetricReceiver) ListenAndReceiveContext(ctx context.Context) error {
	addr := r.Addr
	if addr == "" {
		addr = DefaultMetricsAddr
	}
	var listener net.ListenConfig
	c, err := listener.ListenPacket(ctx, "udp", addr)
	if err != nil {
		return err
	}
	r.ReceiveContext(ctx, c)
	return c.Close()
}

// ReceiveContext accepts incoming datagrams on c and calls
// r.Handler.HandleMetric() for each line in the datagram that successfully
// parses in to a Metric.
//
// If any errors occur reading the UDP stream, they will be emitted as metrics
// that have a bucket name of "error", and a tag that contains the error text.
func (r *MetricReceiver) ReceiveContext(ctx context.Context, c net.PacketConn) {
	msg := make([]byte, 1024)
	for ctx.Err() == nil {
		nbytes, addr, err := c.ReadFrom(msg)
		if err != nil {
			r.handleError(err)
			continue
		}
		buf := make([]byte, nbytes)
		copy(buf, msg[:nbytes])
		go r.handleMessage(addr, buf)
	}
}

func (r *MetricReceiver) handleError(err error) {
	metric := Metric{
		Type:   COUNTER,
		Bucket: fmt.Sprintf("error"),
		Value:  1,
		Tags: map[string]string{
			"error": err.Error(),
		},
	}
	r.Handler.HandleMetric(metric)
}

// Receive is deprecated; use ReceiveContext.
func (r *MetricReceiver) Receive(c net.PacketConn) error {
	defer c.Close()

	msg := make([]byte, 1024)
	for {
		nbytes, addr, err := c.ReadFrom(msg)
		if err != nil {
			continue
		}
		buf := make([]byte, nbytes)
		copy(buf, msg[:nbytes])
		go r.handleMessage(addr, buf)
	}
	panic("not reached")
}

// handleMessage handles the contents of a datagram and attempts to parse a Metric from each line
func (r *MetricReceiver) handleMessage(addr net.Addr, msg []byte) {
	buf := bytes.NewBuffer(msg)
	for {
		line, readerr := buf.ReadBytes('\n')

		// protocol does not require line to end in \n, if EOF use received line if valid
		if readerr != nil && readerr != io.EOF {
			r.handleError(fmt.Errorf("error reading message from %s: %s", addr, readerr))
			return
		} else if readerr != io.EOF {
			// remove newline, only if not EOF
			if len(line) > 0 {
				line = line[:len(line)-1]
			}
		}

		// Only process lines with more than one character
		if len(line) > 1 {
			metric, err := parseLine(line)
			if err != nil {
				r.handleError(fmt.Errorf("error parsing line %q from %s: %s", line, addr, err))
				continue
			}
			go r.Handler.HandleMetric(metric)
		}

		if readerr == io.EOF {
			// if was EOF, finished handling
			return
		}
	}
}

func parseLine(line []byte) (Metric, error) {
	var metric Metric

	buf := bytes.NewBuffer(line)
	bucketAndTags, err := buf.ReadBytes(':')
	if err != nil {
		return metric, fmt.Errorf("error parsing metric name: %s", err)
	}
	// discard trailing ':'
	bucketAndTags = bucketAndTags[:len(bucketAndTags)-1]

	// Parse any 'tags' found in the bucket name. This is a concept created by
	// influxdata and is not present in the statsd spec. Nevertheless, it is
	// widely used.
	tags := bytes.Split(bucketAndTags, []byte{','})
	bucket := tags[0]
	if len(tags) > 1 {
		metric.Tags = make(map[string]string, len(tags)-1)
		for _, tag := range tags {
			tag = bytes.TrimSpace(tag)
			kv := bytes.Split(tag, []byte{'='})
			if len(kv) != 2 {
				continue
			}
			metric.Tags[string(kv[0])] = string(kv[1])
		}
	}
	metric.Bucket = string(bucket)

	value, err := buf.ReadBytes('|')
	if err != nil {
		return metric, fmt.Errorf("error parsing metric value: %s", err)
	}
	metric.Value, err = strconv.ParseFloat(string(value[:len(value)-1]), 64)
	if err != nil {
		return metric, fmt.Errorf("error converting metric value: %s", err)
	}

	metricType := buf.Bytes()
	if err != nil && err != io.EOF {
		return metric, fmt.Errorf("error parsing metric type: %s", err)
	}

	switch string(metricType[:len(metricType)]) {
	case "ms":
		// Timer
		metric.Type = TIMER
	case "g":
		// Gauge
		metric.Type = GAUGE
	case "c":
		metric.Type = COUNTER
	default:
		err = fmt.Errorf("invalid metric type: %q", metricType)
		return metric, err
	}

	return metric, nil
}
