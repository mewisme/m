package integration_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHistoryTimelineAfterMutations(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "history-timeline",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	if code, out := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatalf("add exit=%d out=%s", code, out)
	}

	code, out := runM(t, projDir, cfgPath, "history")
	if code != 0 {
		t.Fatalf("history exit=%d out=%s", code, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 history entries, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "000002") {
		t.Fatalf("newest entry should be 000002, got %q", lines[0])
	}
	if !strings.Contains(out, "graph changed") {
		t.Fatalf("expected graph changed delta, out=%q", out)
	}

	code, out = runM(t, projDir, cfgPath, "history", "--json")
	if code != 0 {
		t.Fatalf("history --json exit=%d out=%s", code, out)
	}
	var entries []struct {
		ID          string `json:"id"`
		GraphDigest string `json:"graphDigest"`
		Delta       string `json:"delta"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &entries); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if len(entries) < 2 {
		t.Fatalf("expected >=2 json entries, got %d", len(entries))
	}
	if entries[0].Delta != "graph changed" {
		t.Fatalf("newest delta=%q", entries[0].Delta)
	}
	if entries[len(entries)-1].Delta != "initial" {
		t.Fatalf("oldest delta=%q", entries[len(entries)-1].Delta)
	}
}

func TestHistoryRestoreStillWorks(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "history-restore",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	if code, out := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatalf("add exit=%d out=%s", code, out)
	}
	if code, out := runM(t, projDir, cfgPath, "history"); code != 0 {
		t.Fatalf("history exit=%d out=%s", code, out)
	}
	if code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "000001"); code != 0 {
		t.Fatalf("restore exit=%d out=%s", code, out)
	}
	if hasDirectDep(t, projDir, "pkg-c") {
		t.Fatal("pkg-c should be removed after restore to first snapshot")
	}
}
