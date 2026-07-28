package lifecycle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/lifecycle"
)

func TestAppendAuditRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	if err := lifecycle.AppendAudit(path, lifecycle.AuditEntry{
		Package: "lodash", Script: "postinstall", ExitCode: 0, DurationMs: 12,
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := lifecycle.ReadAudit(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Package != "lodash" || entries[0].ExitCode != 0 {
		t.Fatalf("got %+v", entries)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if contains(string(raw), "NPM_TOKEN") {
		t.Fatal("audit must not contain secrets")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
