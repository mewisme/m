package conformance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/mewisme/mew/internal/pack"
	"github.com/mewisme/mew/internal/testkit"
)

func TestPackFileListMatchesNpmDryRun(t *testing.T) {
	testkit.RequirePM(t, "npm")
	root := moduleRoot(t)
	fixture := filepath.Join(root, "fixtures", "pack", "minimal-package")
	pkgJSON, err := os.ReadFile(filepath.Join(fixture, "package.json"))
	if err != nil {
		t.Fatal(err)
	}

	mewFiles, err := pack.ListFiles(fixture, pkgJSON)
	if err != nil {
		t.Fatal(err)
	}

	res := testkit.RunPM(t.Context(), "npm", []string{"pack", "--dry-run", "--json"}, fixture, nil)
	if res.ExitCode != 0 {
		t.Fatalf("npm pack exit=%d stderr=%s stdout=%s", res.ExitCode, res.Stderr, res.Stdout)
	}
	var npmOut []struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &npmOut); err != nil {
		t.Fatal(err)
	}
	if len(npmOut) != 1 {
		t.Fatalf("unexpected npm output entries: %d", len(npmOut))
	}
	npmFiles := make([]string, 0, len(npmOut[0].Files))
	for _, f := range npmOut[0].Files {
		npmFiles = append(npmFiles, f.Path)
	}
	sort.Strings(npmFiles)
	sort.Strings(mewFiles)

	// npm may include paths ignored by Mew when package.json files whitelists a parent
	// directory; require Mew's list to match the repository golden and be a subset of npm.
	goldenPath := filepath.Join(root, "testdata", "pack", "minimal-package-files.json")
	wantRaw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if err := json.Unmarshal(wantRaw, &want); err != nil {
		t.Fatal(err)
	}
	if len(mewFiles) != len(want) {
		t.Fatalf("mew files %v want golden %v", mewFiles, want)
	}
	for i := range want {
		if mewFiles[i] != want[i] {
			t.Fatalf("mew files[%d]: got %q want %q", i, mewFiles[i], want[i])
		}
	}

	npmSet := make(map[string]struct{}, len(npmFiles))
	for _, f := range npmFiles {
		npmSet[f] = struct{}{}
	}
	for _, f := range mewFiles {
		if _, ok := npmSet[f]; !ok {
			t.Fatalf("mew file %q missing from npm pack --dry-run list %v", f, npmFiles)
		}
	}
}

func TestPackSimpleFixtureMatchesNpmDryRun(t *testing.T) {
	testkit.RequirePM(t, "npm")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"pack-conformance","version":"2.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte("module.exports = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("skip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".npmignore"), []byte("skip.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	pkgJSON, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	mewFiles, err := pack.ListFiles(dir, pkgJSON)
	if err != nil {
		t.Fatal(err)
	}

	res := testkit.RunPM(t.Context(), "npm", []string{"pack", "--dry-run", "--json"}, dir, nil)
	if res.ExitCode != 0 {
		t.Fatalf("npm pack exit=%d stderr=%s stdout=%s", res.ExitCode, res.Stderr, res.Stdout)
	}
	var npmOut []struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &npmOut); err != nil {
		t.Fatal(err)
	}
	npmFiles := make([]string, 0, len(npmOut[0].Files))
	for _, f := range npmOut[0].Files {
		npmFiles = append(npmFiles, f.Path)
	}
	sort.Strings(npmFiles)
	sort.Strings(mewFiles)
	if len(mewFiles) != len(npmFiles) {
		t.Fatalf("file count: mew %v npm %v", mewFiles, npmFiles)
	}
	for i := range npmFiles {
		if mewFiles[i] != npmFiles[i] {
			t.Fatalf("files[%d]: mew %q npm %q", i, mewFiles[i], npmFiles[i])
		}
	}
}
