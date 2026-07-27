package planner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/linker/planner"
)

func TestProjectMutationDoesNotShareStoreInode(t *testing.T) {
	src := filepath.Join(t.TempDir(), "store-pkg")
	dest := filepath.Join(t.TempDir(), "node_modules", "pkg")
	if err := os.MkdirAll(filepath.Join(src, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(src, "lib", "index.js")
	if err := os.WriteFile(file, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	caps := planner.Capabilities{SameVolume: true, Hardlink: true, Reflink: false}
	op := planner.PlanPackageLink(src, dest, caps)
	if op.Kind == linker.OpHardlink {
		t.Fatal("project-facing package link must not use hardlink")
	}
	if err := linker.Apply(t.Context(), &linker.Plan{Ops: []linker.Op{op}}); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(dest, "lib", "index.js")
	if err := os.WriteFile(linked, []byte("mutated"), 0o644); err != nil {
		t.Fatal(err)
	}
	storeData, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(storeData) != "original" {
		t.Fatalf("store content changed after project mutation: %q", storeData)
	}
}
