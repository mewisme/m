package testkit

import (
	"errors"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"
)

// FaultyRoundTripper fails after After successful requests (0 = fail first).
type FaultyRoundTripper struct {
	After int
	Inner http.RoundTripper
	Err   error
	count atomic.Int64
}

// RoundTrip implements http.RoundTripper.
func (f *FaultyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	n := f.count.Add(1)
	if int(n) > f.After {
		if f.Err != nil {
			return nil, f.Err
		}
		return nil, errors.New("testkit: injected network cut")
	}
	inner := f.Inner
	if inner == nil {
		inner = http.DefaultTransport
	}
	return inner.RoundTrip(req)
}

// LimitedWriter returns ENOSPC after N bytes are written.
type LimitedWriter struct {
	N     int64
	W     io.Writer
	wrote int64
}

// Write implements io.Writer.
func (l *LimitedWriter) Write(p []byte) (int, error) {
	if l.wrote >= l.N {
		return 0, &os.PathError{Op: "write", Path: "limited", Err: syscall.ENOSPC}
	}
	remain := l.N - l.wrote
	chunk := p
	truncated := false
	if int64(len(chunk)) > remain {
		chunk = p[:remain]
		truncated = true
	}
	w := l.W
	if w == nil {
		w = io.Discard
	}
	n, err := w.Write(chunk)
	l.wrote += int64(n)
	if err != nil {
		return n, err
	}
	if truncated || l.wrote >= l.N {
		return n, &os.PathError{Op: "write", Path: "limited", Err: syscall.ENOSPC}
	}
	return n, nil
}
