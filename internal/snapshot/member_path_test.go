package snapshot_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/snapshot"
)

func TestParseMemberManifestPathAccepts(t *testing.T) {
	for _, rel := range []string{
		"packages/alpha/package.json",
		"packages/beta/package.json",
		"nested/pkg/package.json",
	} {
		id, err := snapshot.ParseMemberManifestPath(rel)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		want := graph.ImporterID(filepath.ToSlash(filepath.Dir(rel)))
		if id != want {
			t.Fatalf("%s: got %q want %q", rel, id, want)
		}
	}
}

func TestParseMemberManifestPathRejects(t *testing.T) {
	for _, rel := range []string{
		"",
		"package.json",
		"../packages/alpha/package.json",
		"packages/../alpha/package.json",
		".mew/cache/package.json",
		".git/hooks/package.json",
		"/etc/passwd/package.json",
		"packages/alpha",
		"packages/alpha/other.json",
	} {
		if _, err := snapshot.ParseMemberManifestPath(rel); err == nil {
			t.Fatalf("expected error for %q", rel)
		}
	}
}

func TestParseMemberManifestPathWindowsCaseCollision(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only case collision check")
	}
	a, err := snapshot.ParseMemberManifestPath("Packages/Alpha/package.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err := snapshot.ParseMemberManifestPath("packages/alpha/package.json")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("case-fold mismatch: %q vs %q", a, b)
	}
}
