package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/testkit"
	"github.com/mewisme/m/internal/workspace"
)

func TestExpandSimple(t *testing.T) {
	root := filepath.Join(testkit.ModuleRoot(t), "fixtures", "projects", "workspace-simple")
	members, err := workspace.Expand(root, []string{"packages/*"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"packages/a", "packages/b"}
	if len(members) != len(want) {
		t.Fatalf("got %v want %v", members, want)
	}
	for i := range want {
		if members[i] != want[i] {
			t.Fatalf("got %v want %v", members, want)
		}
	}
}

func TestExpandBracesAndNegation(t *testing.T) {
	root := filepath.Join(testkit.ModuleRoot(t), "fixtures", "projects", "workspace-nested")
	idx, err := workspace.BuildIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"apps/web", "packages/core"}
	if len(idx.Members) != len(want) {
		t.Fatalf("got %v want %v", idx.Members, want)
	}
	for i := range want {
		if idx.Members[i] != want[i] {
			t.Fatalf("got %v want %v", idx.Members, want)
		}
	}
}

func TestCycleRejected(t *testing.T) {
	dir := t.TempDir()
	rootPkg := []byte(`{"name":"root","private":true,"workspaces":["packages/*"]}`)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), rootPkg, 0o644); err != nil {
		t.Fatal(err)
	}
	mem := filepath.Join(dir, "packages", "a")
	if err := os.MkdirAll(mem, 0o755); err != nil {
		t.Fatal(err)
	}
	// Member claims workspaces that include the repo root via ..
	bad := []byte(`{"name":"a","version":"1.0.0","workspaces":["../.."]}`)
	if err := os.WriteFile(filepath.Join(mem, "package.json"), bad, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := workspace.BuildIndex(dir)
	if err == nil {
		t.Fatal("expected cycle")
	}
	if apperr.CodeOf(err) != apperr.Manifest {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}
