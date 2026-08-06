package archcheck_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRuntimeCertificationEvidence verifies runtime certification evidence
// is consistent with plan status and free of stale claims.
func TestRuntimeCertificationEvidence(t *testing.T) {
	root := repoRoot(t)

	// 1. Evidence document must exist and report GREEN.
	evPath := filepath.Join(root, "docs", "evidence", "runtime", "0050-0051-certification.md")
	ev, err := os.ReadFile(evPath)
	if err != nil {
		t.Fatalf("cannot read runtime certification evidence: %v", err)
	}
	evText := string(ev)

	if !strings.Contains(evText, "**GREEN**") {
		t.Error("runtime certification evidence does not report GREEN status")
	}
	if !strings.Contains(evText, "`6831061") {
		t.Error("runtime certification evidence missing certified commit SHA (6831061...)")
	}
	if !strings.Contains(evText, "Normal CI") || !strings.Contains(evText, "Full certification") {
		t.Error("runtime certification evidence missing CI run references")
	}

	// 2. All 12 OS/Node combinations present (3 OS x 4 Node).
	for _, combo := range []string{
		"Linux (amd64)", "macOS (arm64)", "Windows (amd64)",
		"18.x", "20.x", "22.x", "24.x",
	} {
		if !strings.Contains(evText, combo) {
			t.Errorf("runtime certification evidence missing platform/version: %s", combo)
		}
	}

	// 3. Status.json: 0050/0051 completed, 0052 current.
	st, err := os.ReadFile(filepath.Join(root, "plans", "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	stText := string(st)
	if !strings.Contains(stText, `"0050"`) || !strings.Contains(stText, `"0051"`) {
		t.Error("status.json missing 0050 or 0051 in completedMvps")
	}
	if !strings.Contains(stText, `"currentMvp": "0052"`) {
		t.Error("status.json does not show 0052 as currentMvp")
	}
	if !strings.Contains(stText, `"lastCertifiedCoreCommit": "6831061`) {
		t.Error("status.json lastCertifiedCoreCommit does not match evidence SHA")
	}

	// 4. CHECKLIST header agrees with status.json.
	cl, err := os.ReadFile(filepath.Join(root, "plans", "CHECKLIST.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cl), "Last certified core commit: `6831061") {
		t.Error("CHECKLIST.md header does not match status.json certified commit SHA")
	}

	// 5. No stale limitation text.
	for _, phrase := range []string{
		"macOS and Windows certification pending",
		"node-args + TypeScript",
		"node-args subcommand does not",
	} {
		if strings.Contains(evText, phrase) {
			t.Errorf("runtime certification evidence contains stale limitation: %q", phrase)
		}
	}
}
