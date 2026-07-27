package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/config"
)

func TestPrepareAddDependencySetsRange(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"add-prepare","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: root, Config: &config.Effective{}}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	proj, err := sess.ReopenProject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	opts := InstallOptions{AddSpec: "lodash@4.17.21", WriteManifest: true}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	if err := prepareAddDependency(ctx, sac, proj, &opts); err != nil {
		t.Fatal(err)
	}
	got, ok := proj.Doc.Dependencies["lodash"]
	if !ok {
		t.Fatal("lodash not added")
	}
	if got != "^4.17.21" {
		t.Fatalf("range=%q", got)
	}
}

func TestPrepareAddDependencySaveExact(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"add-exact","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: root, Config: &config.Effective{}}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	proj, err := sess.ReopenProject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	opts := InstallOptions{AddSpec: "lodash@4.17.21", AddSaveExact: true, WriteManifest: true}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	if err := prepareAddDependency(ctx, sac, proj, &opts); err != nil {
		t.Fatal(err)
	}
	got := proj.Doc.Dependencies["lodash"]
	if got != "4.17.21" {
		t.Fatalf("exact=%q", got)
	}
}
