package runner_test

import (
	"os"
	"testing"
)

func TestTTYProbeInteractive(t *testing.T) {
	if os.Getenv("MEW_CONFORMANCE_TTY") != "1" {
		t.Skip("probe requires MEW_CONFORMANCE_TTY=1")
	}
	if !isTerminal(os.Stdout) {
		t.Skip("probe requires TTY")
	}
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
