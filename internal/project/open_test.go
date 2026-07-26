package project_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/testkit"
)

func TestOpenWalkUp(t *testing.T) {
	root := testkit.ModuleRoot(t)
	base := filepath.Join(root, "fixtures", "projects", "basic-cjs")
	sub := filepath.Join(base, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := project.Open(context.Background(), sub)
	if err != nil {
		t.Fatal(err)
	}
	if p.Doc == nil || p.Doc.Name != "basic-cjs" {
		t.Fatalf("%+v", p.Doc)
	}
	if p.Normalized == nil {
		t.Fatal("nil normalized")
	}
}

func TestOpenAtMember(t *testing.T) {
	root := filepath.Join(testkit.ModuleRoot(t), "fixtures", "projects", "workspace-simple")
	p, err := project.OpenAt(context.Background(), root, "packages/a")
	if err != nil {
		t.Fatal(err)
	}
	if p.Doc.Name != "a" {
		t.Fatalf("name=%q", p.Doc.Name)
	}
	if p.Rel != "packages/a" {
		t.Fatalf("rel=%q", p.Rel)
	}
}

func TestFindRootNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := project.FindRoot(dir)
	if err == nil {
		t.Fatal("expected not found")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}
