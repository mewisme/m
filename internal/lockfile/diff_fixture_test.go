package lockfile_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func TestLockRevisionFixtureDiff(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "diff", "lock-revisions")
	beforePath := filepath.Join(root, "before.m.lock")
	afterPath := filepath.Join(root, "after.m.lock")
	expectedPath := filepath.Join(root, "expected.json")

	before, err := readMlockGraph(t, beforePath)
	if err != nil {
		t.Fatal(err)
	}
	after, err := readMlockGraph(t, afterPath)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := lockfile.DiffGraphs(before, after)
	if err != nil {
		t.Fatal(err)
	}
	got, err := lockfile.EncodeDiffJSON(diff)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	var gotDoc, wantDoc graphDiffJSON
	if err := json.Unmarshal(got, &gotDoc); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &wantDoc); err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(gotDoc, wantDoc) {
		t.Fatalf("diff mismatch\ngot=%s\nwant=%s", got, want)
	}
}

type graphDiffJSON struct {
	PackagesAdded   []string                         `json:"packagesAdded"`
	PackagesRemoved []string                         `json:"packagesRemoved"`
	Specifiers      []lockfile.ImporterSpecifierDiff `json:"specifiers"`
}

func readMlockGraph(t *testing.T, path string) (*graph.Graph, error) {
	t.Helper()
	adapter := lockfile.AdapterFor(project.IdentityMew)
	if adapter == nil {
		t.Fatal("m.lock adapter missing")
	}
	return adapter.Read(context.Background(), path)
}

func jsonEqual(a, b graphDiffJSON) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
