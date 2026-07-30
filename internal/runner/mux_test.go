package runner_test

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/mewisme/mew/internal/runner"
)

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestPrefixWriterNoTrailingNewline(t *testing.T) {
	var out bytes.Buffer
	w := runner.NewPrefixWriter("pkg", "build", "stdout", &out, nil, false)
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	want := "[pkg] hello\n"
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
}

func TestPrefixWriterCRLF(t *testing.T) {
	var out bytes.Buffer
	w := runner.NewPrefixWriter("pkg", "build", "stdout", &out, nil, false)
	if _, err := w.Write([]byte("a\r\nb\r")); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[pkg] a\n[pkg] b\n") {
		t.Fatalf("got %q", out.String())
	}
}

func TestPrefixWriterSplitUTF8(t *testing.T) {
	var out bytes.Buffer
	w := runner.NewPrefixWriter("pkg", "build", "stdout", &out, nil, false)
	r := strings.Repeat("é", 1)
	b := []byte(r)
	if _, err := w.Write(b[:1]); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(b[1:]); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "é") {
		t.Fatalf("got %q", out.String())
	}
}

func TestPrefixWriterVeryLongLine(t *testing.T) {
	var out bytes.Buffer
	w := runner.NewPrefixWriter("pkg", "build", "stdout", &out, nil, false)
	long := strings.Repeat("x", 1<<20+10)
	if _, err := w.Write([]byte(long)); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[mew: output truncated]") {
		t.Fatal("expected truncation marker")
	}
}

func TestPrefixWriterConcurrentWriters(t *testing.T) {
	var out syncBuffer
	w := runner.NewPrefixWriter("pkg", "build", "stdout", &out, nil, false)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = w.Write([]byte("line\n"))
		}(i)
	}
	wg.Wait()
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "[pkg]") < 8 {
		t.Fatalf("lines=%d", strings.Count(out.String(), "[pkg]"))
	}
}

func TestPrefixWriterWriterError(t *testing.T) {
	w := runner.NewPrefixWriter("pkg", "build", "stdout", errWriter{}, nil, false)
	if _, err := w.Write([]byte("x\n")); err == nil {
		t.Fatal("expected write error")
	}
}

func TestAggregateBufferSpill(t *testing.T) {
	dir := t.TempDir()
	buf := runner.NewAggregateBuffer(dir, "task-0", "stdout")
	chunk := bytes.Repeat([]byte("a"), 300*1024)
	if _, err := buf.Write(chunk); err != nil {
		t.Fatal(err)
	}
	got, err := buf.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(chunk) {
		t.Fatalf("len=%d want %d", len(got), len(chunk))
	}
	if err := buf.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAggregateBufferHardCap(t *testing.T) {
	dir := t.TempDir()
	buf := runner.NewAggregateBuffer(dir, "task-1", "stderr")
	chunk := bytes.Repeat([]byte("b"), 17*1024*1024)
	if _, err := buf.Write(chunk); err != nil {
		t.Fatal(err)
	}
	got, err := buf.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > 16*1024*1024+len("[mew: output truncated]") {
		t.Fatalf("len=%d exceeds cap", len(got))
	}
	if err := buf.Close(); err != nil {
		t.Fatal(err)
	}
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestUTF8FullRune(t *testing.T) {
	b := []byte{0xC3}
	if utf8.FullRune(b) {
		t.Fatal("partial rune should not be full")
	}
}
