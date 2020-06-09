package statsd

import (
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
