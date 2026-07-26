package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectInfoAndPkgGet(t *testing.T) {
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(modRoot, "fixtures", "projects", "workspace-simple")

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--cwd", fixture, "project", "info", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["name"] != "workspace-simple" {
		t.Fatalf("%v", doc)
	}
	members, _ := doc["members"].([]any)
	if len(members) < 2 {
		t.Fatalf("members=%v", members)
	}

	root = NewMRoot(testBuildInfo())
	buf.Reset()
	root.SetOut(buf)
	root.SetErr(buf)
	pkgFix := filepath.Join(modRoot, "fixtures", "projects", "basic-cjs")
	root.SetArgs([]string{"--cwd", pkgFix, "pkg", "get", "name"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "basic-cjs" {
		t.Fatalf("%q", buf.String())
	}
}
