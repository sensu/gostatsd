package statsd

import (
	"context"
	"net"
	"reflect"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := map[string]Metric{
		"foo.bar.baz:2|c":       Metric{Bucket: "foo.bar.baz", Value: 2.0, Type: COUNTER},
		"abc.def.g:3|g":         Metric{Bucket: "abc.def.g", Value: 3, Type: GAUGE},
		"def.g:10|ms":           Metric{Bucket: "def.g", Value: 10, Type: TIMER},
		"asdf,x=y,foo=bar:10|c": Metric{Bucket: "asdf", Value: 10, Type: COUNTER, Tags: map[string]string{"x": "y", "foo": "bar"}},
		"asdf,asdf,x=y:1|c":     Metric{Bucket: "asdf", Value: 1, Type: COUNTER, Tags: map[string]string{"x": "y"}},
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			result, err := parseLine([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result, expected) {
				t.Errorf("expected %s, got %s", expected, result)
			}
		})
	}

	failing := []string{"fOO|bar:bazkk", "foo.bar.baz:1|q"}
	for _, tc := range failing {
		t.Run(tc, func(t *testing.T) {
			result, err := parseLine([]byte(tc))
			if err == nil {
				t.Errorf("expected error but got %s", result)
			}
		})
	}
}

type testHandler struct {
	t testing.TB
}

func (t testHandler) HandleMetric(m Metric) {
	if m.Type < COUNTER || m.Type > GAUGE {
		t.t.Errorf("bad metric: %v", m)
	}
}

func TestReceiveContext(t *testing.T) {
	var server MetricReceiver
	server.Handler = testHandler{t}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var listener net.ListenConfig
	c, err := listener.ListenPacket(ctx, "udp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	go server.ReceiveContext(ctx, c)
	tests := []string{
		"foo.bar.baz:2|c",
		"abc.def.g:3|g",
		"def.g:10|ms",
		"asdf,x=y,foo=bar:10|c",
		"asdf,asdf,x=y:1|c",
	}
	client, err := net.Dial("udp", c.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			n, err := client.Write([]byte(test))
			if err != nil {
				t.Fatal(err)
			}
			if got, want := n, len(test); got != want {
				t.Errorf("bad number of bytes written: got %d, want %d", got, want)
			}
		})
	}
}
