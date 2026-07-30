package presentation

import (
	"io"
	"os"
)

// WriterPair returns stdout/stderr writers, defaulting to os stdout/stderr.
func WriterPair(out, err io.Writer) StreamWriters {
	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}
	return StreamWriters{Out: out, Err: err}
}
