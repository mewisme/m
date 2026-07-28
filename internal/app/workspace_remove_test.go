package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
)

func TestPrepareFilteredRemoveUpdatesMemberOnly(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_WORKSPACES", "1")
	root := t.TempDir()
	writeTestPkg(t, root, `{"name":"root","workspaces":["packages/*"]}`)
	writeTestPkg(t, filepath.Join(root, "packages", "alpha"), `{"name":"alpha","dependencies":{"lodash":"^4.0.0"}}`)
	writeTestPkg(t, filepath.Join(root, "packages", "beta"), `{"name":"beta","dependencies":{"lodash":"^4.0.0"}}`)

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
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	opts := InstallOptions{Filter: []string{"alpha"}, WriteManifest: true}
	if err := prepareFilteredRemove(ctx, sac, proj, &opts, "lodash"); err != nil {
		t.Fatal(err)
	}
	if _, ok := opts.MemberEdits["packages/alpha"]; !ok {
		t.Fatal("alpha member edit missing")
	}
	if _, ok := opts.MemberEdits["packages/beta"]; ok {
		t.Fatal("beta should not be edited")
	}
	betaData, err := os.ReadFile(filepath.Join(root, "packages", "beta", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstr(string(betaData), "lodash") {
		t.Fatal("beta package.json should be unchanged on disk")
	}
}

func TestPrepareFilteredRemoveNotFound(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_WORKSPACES", "1")
	root := t.TempDir()
	writeTestPkg(t, root, `{"name":"root","workspaces":["packages/*"]}`)
	writeTestPkg(t, filepath.Join(root, "packages", "alpha"), `{"name":"alpha"}`)

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
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	opts := InstallOptions{Filter: []string{"alpha"}}
	err = prepareFilteredRemove(ctx, sac, proj, &opts, "missing-pkg")
	if err == nil || apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func writeTestPkg(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsSubstr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexSubstr(s, sub) >= 0)
}

func indexSubstr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
