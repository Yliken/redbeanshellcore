package httpform

import (
	"errors"
	"strings"
	"testing"
)

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestReadBodyLimited(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		limit     int64
		wantBody  string
		oversized bool
	}{
		{name: "below", body: "123", limit: 4, wantBody: "123"},
		{name: "exact", body: "1234", limit: 4, wantBody: "1234"},
		{name: "over", body: "12345", limit: 4, wantBody: "1234", oversized: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, oversized, err := readBodyLimited(strings.NewReader(test.body), test.limit)
			if err != nil || string(body) != test.wantBody || oversized != test.oversized {
				t.Fatalf("body=%q oversized=%v err=%v", body, oversized, err)
			}
		})
	}
}

func TestReadBodyLimitedPreservesReadError(t *testing.T) {
	sentinel := errors.New("read failed")
	_, _, err := readBodyLimited(failingReader{err: sentinel}, 4)
	if !errors.Is(err, sentinel) {
		t.Fatalf("应保留读取错误，got %v", err)
	}
}
