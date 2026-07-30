package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLockDiffFromToHuman(t *testing.T) {
	projDir := t.TempDir()
	fixtureRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "diff", "lock-revisions"))
	if err != nil {
		t.Fatal(err)
	}
	before := filepath.Join(fixtureRoot, "before.m.lock")
	after := filepath.Join(fixtureRoot, "after.m.lock")
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "lock-diff-from-to",
  "version": "1.0.0"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "lock", "diff", "--from", before, "--to", after)
	if code != 0 {
		t.Fatalf("lock diff exit=%d out=%s", code, out)
	}
	for _, want := range []string{"+ a  2.0.0", "- a  1.0.0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestLockDiffAliasJSON(t *testing.T) {
	projDir := t.TempDir()
	fixtureRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "diff", "lock-revisions"))
	if err != nil {
		t.Fatal(err)
	}
	before := filepath.Join(fixtureRoot, "before.m.lock")
	after := filepath.Join(fixtureRoot, "after.m.lock")
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "lock-diff-alias",
  "version": "1.0.0"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "diff", "lock", "--from", before, "--to", after, "--json")
	if code != 0 {
		t.Fatalf("diff lock exit=%d out=%s", code, out)
	}
	var doc struct {
		PackagesAdded   []string `json:"packagesAdded"`
		PackagesRemoved []string `json:"packagesRemoved"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &doc); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if len(doc.PackagesAdded) != 1 || doc.PackagesAdded[0] != "a@2.0.0" {
		t.Fatalf("added: %v", doc.PackagesAdded)
	}
	if len(doc.PackagesRemoved) != 1 || doc.PackagesRemoved[0] != "a@1.0.0" {
		t.Fatalf("removed: %v", doc.PackagesRemoved)
	}
}

func TestLockDiffIncumbentAgainstOtherLock(t *testing.T) {
	projDir := t.TempDir()
	fixtureRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "diff", "lock-revisions"))
	if err != nil {
		t.Fatal(err)
	}
	before := filepath.Join(fixtureRoot, "before.m.lock")
	after := filepath.Join(fixtureRoot, "after.m.lock")
	beforeBytes, err := os.ReadFile(before)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "lock-diff-incumbent",
  "version": "1.0.0",
  "packageManager": "mew@0.0.0"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "m.lock"), beforeBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "lock", "diff", after)
	if code != 0 {
		t.Fatalf("lock diff exit=%d out=%s", code, out)
	}
	for _, want := range []string{"+ a  2.0.0", "- a  1.0.0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
