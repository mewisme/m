package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func setupDoctorGreenfield(t *testing.T) (projDir, cfgPath string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	projDir = t.TempDir()
	testkit.CopyFixture(t, "projects/mlock-greenfield", projDir)

	cfgPath = filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return projDir, cfgPath
}

func TestDoctorGreenfieldJSON(t *testing.T) {
	projDir, cfgPath := setupDoctorGreenfield(t)
	code, out := runM(t, projDir, cfgPath, "doctor", "--json")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	var doc struct {
		SchemaVersion int  `json:"schemaVersion"`
		OK            bool `json:"ok"`
		Checks        []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	if doc.SchemaVersion != 1 || !doc.OK {
		t.Fatalf("doc=%+v", doc)
	}
	seen := map[string]string{}
	for _, c := range doc.Checks {
		seen[c.ID] = c.Status
	}
	for _, id := range []string{"project", "config", "cache", "store", "lock", "filesystem", "transaction"} {
		if seen[id] == "" {
			t.Fatalf("missing check %q: %+v", id, doc.Checks)
		}
	}
	if seen["project"] != "ok" || seen["lock"] != "ok" {
		t.Fatalf("checks=%v", seen)
	}
}

func TestDoctorFailsOutsideProject(t *testing.T) {
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _ := runM(t, projDir, cfgPath, "doctor")
	if code == 0 {
		t.Fatal("expected failure outside project")
	}
}
