package presentation

import (
	"strings"
	"testing"
)

func TestLiveInstallEnvironSuppressesModeQueries(t *testing.T) {
	env := liveInstallEnviron([]string{
		"PATH=/bin",
		"WT_SESSION=abc",
		"TERM_PROGRAM=ghostty",
		"SSH_TTY=/dev/pts/0",
	})
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "WT_SESSION=") {
		t.Fatalf("WT_SESSION must be stripped: %q", joined)
	}
	if !strings.Contains(joined, "SSH_TTY=1") {
		t.Fatalf("missing SSH_TTY spoof: %q", joined)
	}
	if !strings.Contains(joined, "TERM_PROGRAM=Apple_Terminal") {
		t.Fatalf("missing Apple_Terminal spoof: %q", joined)
	}
	if !strings.Contains(joined, "PATH=/bin") {
		t.Fatalf("base env dropped: %q", joined)
	}
}
