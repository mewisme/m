package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestOutdatedJSONReportsUpdates(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/outdated-tree", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	code, out := runM(t, projDir, cfgPath, "outdated", "--json")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	var entries []map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &entries); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	if len(entries) == 0 {
		t.Fatalf("expected outdated entries, out=%s", out)
	}
	found := false
	for _, e := range entries {
		if e["package"] == "pkg-b" {
			found = true
			for _, k := range []string{"current", "wanted", "latest"} {
				if e[k] == "" {
					t.Fatalf("missing %s in %v", k, e)
				}
			}
			if e["current"] == e["wanted"] && e["current"] == e["latest"] {
				t.Fatalf("pkg-b not outdated: %v", e)
			}
		}
	}
	if !found {
		t.Fatalf("pkg-b missing in %v", entries)
	}
}

func TestLsDependencyTreeJSON(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "ls-tree",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	code, out = runM(t, projDir, cfgPath, "ls", "--json")
	if code != 0 {
		t.Fatalf("ls exit=%d out=%s", code, out)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	var deps []map[string]string
	if err := json.Unmarshal(doc["dependencies"], &deps); err != nil {
		t.Fatalf("deps json: %v", err)
	}
	found := false
	for _, d := range deps {
		if d["name"] == "lodash" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("lodash missing in tree: %s", out)
	}
}
