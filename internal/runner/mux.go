package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"unicode/utf8"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

const (
	prefixWriterMaxLine   = 1 << 20 // 1 MiB
	aggregateMemThreshold = 256 * 1024
	aggregateHardCap      = 16 * 1024 * 1024
	aggregateTruncMarker  = "[mew: output truncated]"
)

// PrefixWriter prefixes each complete line with [package] for stream mode.
type PrefixWriter struct {
	mu         sync.Mutex
	prefix     string
	out        io.Writer
	buf        []byte
	err        error
	pkg        string
	script     string
	stream     string
	rep        diagnostics.Reporter
	structured bool
	seq        int
}

// NewPrefixWriter creates a line-buffered prefixed writer.
func NewPrefixWriter(pkg, script, stream string, out io.Writer, rep diagnostics.Reporter, structured bool) *PrefixWriter {
	return &PrefixWriter{
		prefix:     fmt.Sprintf("[%s] ", pkg),
		out:        out,
		pkg:        pkg,
		script:     script,
		stream:     stream,
		rep:        rep,
		structured: structured,
	}
}

func (w *PrefixWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return 0, w.err
	}
	total := len(p)
	w.buf = append(w.buf, p...)
	w.processBuffer(false)
	return total, w.err
}

func (w *PrefixWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.processBuffer(true)
	return w.err
}

func (w *PrefixWriter) Close() error {
	return w.Flush()
}

func (w *PrefixWriter) processBuffer(final bool) {
	for len(w.buf) > 0 {
		if len(w.buf) > prefixWriterMaxLine {
			w.emitLine(w.buf[:prefixWriterMaxLine], false)
			w.buf = w.buf[prefixWriterMaxLine:]
			w.emitLine([]byte(aggregateTruncMarker), false)
			continue
		}
		idx, adv := findLineBreak(w.buf)
		if idx < 0 {
			if !final {
				w.buf = trimPartialUTF8(w.buf)
				return
			}
			if len(w.buf) > 0 && utf8.Valid(w.buf) {
				w.emitLine(w.buf, true)
				w.buf = nil
			}
			return
		}
		line := w.buf[:idx]
		w.buf = w.buf[adv:]
		if len(line) > 0 {
			w.emitLine(line, false)
		}
	}
}

func trimPartialUTF8(b []byte) []byte {
	i := 0
	for i < len(b) {
		_, size := utf8.DecodeRune(b[i:])
		if size == 0 {
			break
		}
		i += size
	}
	return b[:i]
}

func findLineBreak(b []byte) (lineEnd int, advance int) {
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			return i, i + 1
		}
		if b[i] == '\r' {
			if i+1 < len(b) && b[i+1] == '\n' {
				return i, i + 2
			}
			return i, i + 1
		}
	}
	return -1, 0
}

func (w *PrefixWriter) emitLine(line []byte, partial bool) {
	if w.err != nil {
		return
	}
	if w.structured && w.rep != nil {
		w.seq++
		w.rep.ChildOutput(diagnostics.ChildOutputEvent{
			V:       1,
			Type:    "child-output",
			Package: w.pkg,
			Script:  w.script,
			Stream:  w.stream,
			Message: string(line),
			Partial: partial,
			Seq:     &w.seq,
		}, diagnostics.WorkspaceOutputStream)
		return
	}
	if _, err := w.out.Write([]byte(w.prefix)); err != nil {
		w.err = err
		return
	}
	if _, err := w.out.Write(line); err != nil {
		w.err = err
		return
	}
	if _, err := w.out.Write([]byte{'\n'}); err != nil {
		w.err = err
	}
}

// AggregateBuffer collects one stream with spill-to-disk support.
type AggregateBuffer struct {
	mu sync.Mutex

	dir    string
	key    string
	stream string

	mem   []byte
	spill *os.File
	total int
	err   error
}

// NewAggregateBuffer creates a per-stream aggregate collector under dir/key.
func NewAggregateBuffer(dir, key, stream string) *AggregateBuffer {
	return &AggregateBuffer{dir: dir, key: key, stream: stream}
}

func (a *AggregateBuffer) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return 0, a.err
	}
	total := len(p)
	remaining := aggregateHardCap - a.total
	if remaining <= 0 {
		return total, nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	a.total += len(p)

	if len(a.mem) < aggregateMemThreshold {
		space := aggregateMemThreshold - len(a.mem)
		if len(p) <= space {
			a.mem = append(a.mem, p...)
			return total, nil
		}
		a.mem = append(a.mem, p[:space]...)
		p = p[space:]
	}

	if a.spill == nil {
		f, err := a.openSpill()
		if err != nil {
			a.err = err
			return 0, err
		}
		a.spill = f
	}
	if len(p) > 0 {
		if _, err := a.spill.Write(p); err != nil {
			a.err = apperr.Wrap(apperr.IO, "aggregate.spill", a.key, err)
			return 0, a.err
		}
	}
	return total, nil
}

func (a *AggregateBuffer) openSpill() (*os.File, error) {
	dir := filepath.Join(a.dir, a.key)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "aggregate.spill", dir, err)
	}
	name := filepath.Join(dir, a.stream)
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "aggregate.spill", name, err)
	}
	return f, nil
}

// Bytes returns collected bytes, appending truncation marker when capped.
func (a *AggregateBuffer) Bytes() ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return nil, a.err
	}
	out, err := a.readAll()
	if err != nil {
		return nil, err
	}
	if a.total >= aggregateHardCap {
		out = append(out, aggregateTruncMarker...)
	}
	if a.spill != nil {
		name := a.spill.Name()
		_ = a.spill.Close()
		a.spill = nil
		_ = name
	}
	return out, nil
}

func (a *AggregateBuffer) readAll() ([]byte, error) {
	out := append([]byte(nil), a.mem...)
	if a.spill != nil {
		b, err := os.ReadFile(a.spill.Name())
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "aggregate.replay", a.spill.Name(), err)
		}
		out = append(out, b...)
	}
	return out, nil
}

// Close releases spill file handles.
func (a *AggregateBuffer) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.spill != nil {
		return a.spill.Close()
	}
	return nil
}
