package testkit

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// FSCapabilities reports filesystem features available under dir.
type FSCapabilities struct {
	Symlink       bool `json:"symlink"`
	Junction      bool `json:"junction"` // Windows only when supported
	CaseSensitive bool `json:"caseSensitive"`
}

// ProbeFS probes symlink, junction (Windows), and case-sensitivity under dir.
func ProbeFS(t testing.TB, dir string) FSCapabilities {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var caps FSCapabilities

	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err == nil {
		caps.Symlink = true
		_ = os.Remove(link)
	}

	if runtime.GOOS == "windows" {
		// Junction probe via mklink requires privileges; treat symlink success as soft signal.
		// Dedicated junction creation is deferred; report false unless env MEW_TEST_JUNCTION=1 later.
		caps.Junction = false
	}

	lower := filepath.Join(dir, "CaseProbe")
	upper := filepath.Join(dir, "caseprobe")
	if err := os.WriteFile(lower, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(upper); err == nil {
		// On case-insensitive FS, upper path resolves to same file.
		caps.CaseSensitive = false
	} else {
		caps.CaseSensitive = true
	}
	_ = os.Remove(lower)
	_ = os.Remove(target)
	return caps
}
