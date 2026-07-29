package integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type projectSnapshot struct {
	pkgJSON []byte
	lock    []byte
	nmTree  string
}

func snapshotProject(t *testing.T, projDir string) projectSnapshot {
	t.Helper()
	pkgPath := filepath.Join(projDir, "package.json")
	pkgData, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	var lockData []byte
	if data, err := os.ReadFile(filepath.Join(projDir, "m.lock")); err == nil {
		lockData = data
	}
	return projectSnapshot{
		pkgJSON: pkgData,
		lock:    lockData,
		nmTree:  digestTree(t, filepath.Join(projDir, "node_modules")),
	}
}

func assertProjectUnchanged(t *testing.T, projDir string, before projectSnapshot) {
	t.Helper()
	after := snapshotProject(t, projDir)
	if string(after.pkgJSON) != string(before.pkgJSON) {
		t.Fatal("package.json content changed")
	}
	if string(after.lock) != string(before.lock) {
		t.Fatal("m.lock content changed")
	}
	if after.nmTree != before.nmTree {
		t.Fatal("node_modules tree digest changed")
	}
}

func digestTree(t *testing.T, root string) string {
	t.Helper()
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", root)
	}
	var entries []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		entries = append(entries, rel+":"+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	h := sha256.New()
	for _, line := range entries {
		_, _ = io.WriteString(h, line+"\n")
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestExplainPlanDiffHistoryNoMutation(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "nomut-0028",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "4.17.21" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	before := snapshotProject(t, projDir)

	fixtureRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "diff", "lock-revisions"))
	if err != nil {
		t.Fatal(err)
	}
	beforeLock := filepath.Join(fixtureRoot, "before.m.lock")
	afterLock := filepath.Join(fixtureRoot, "after.m.lock")

	cmds := [][]string{
		{"explain", "pkg-a"},
		{"explain", "pkg-a", "--json"},
		{"explain", "peer", "react"},
		{"plan"},
		{"plan", "--json"},
		{"lock", "diff", "--from", beforeLock, "--to", afterLock},
		{"diff", "lock", "--from", beforeLock, "--to", afterLock, "--json"},
		{"history"},
		{"history", "--json"},
	}
	for _, args := range cmds {
		args := args
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, out := runM(t, projDir, cfgPath, args...)
			if code != 0 {
				t.Fatalf("exit=%d out=%s", code, out)
			}
			assertProjectUnchanged(t, projDir, before)
		})
	}
}
