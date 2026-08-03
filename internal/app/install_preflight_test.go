package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/testkit"
)

// preflightCtx builds a project dir plus app context with no network access.
func preflightCtx(t *testing.T, files map[string]string) (string, *Context) {
	t.Helper()
	testkit.CleanEnv(t)
	proj := t.TempDir()
	for name, body := range files {
		full := filepath.Join(proj, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	loadOpts := config.LoadOptions{CWD: proj, ProjectRoot: proj}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}
	return proj, ac
}

// assertNoMewDir is the core Group 9 guarantee: a rejected install leaves no
// transaction state behind because it never began a mutation session.
func assertNoMewDir(t *testing.T, proj string) {
	t.Helper()
	mew := filepath.Join(proj, ".mew")
	if _, err := os.Stat(mew); err == nil {
		entries, _ := os.ReadDir(mew)
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf(".mew created before preflight passed: %v", names)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat .mew: %v", err)
	}
}

// A missing lockfile under --frozen must be rejected without creating .mew.
func TestPreflightFrozenMissingLockNoMewDir(t *testing.T) {
	proj, ac := preflightCtx(t, map[string]string{
		"package.json": `{"name":"p","version":"1.0.0","dependencies":{"ms":"2.1.3"}}`,
	})
	if _, err := Install(context.Background(), ac, InstallOptions{Frozen: true}); err == nil {
		t.Fatal("expected frozen install to fail with no lockfile")
	}
	assertNoMewDir(t, proj)
}

// A corrupt lockfile under --frozen must be rejected without creating .mew.
func TestPreflightFrozenCorruptLockNoMewDir(t *testing.T) {
	proj, ac := preflightCtx(t, map[string]string{
		"package.json": `{"name":"p","version":"1.0.0","dependencies":{"ms":"2.1.3"}}`,
		"m.lock":       "{not valid json",
	})
	if _, err := Install(context.Background(), ac, InstallOptions{Frozen: true}); err == nil {
		t.Fatal("expected frozen install to fail with corrupt lockfile")
	}
	assertNoMewDir(t, proj)
}

// Yarn PnP must be rejected without creating .mew.
func TestPreflightYarnPnPRejectedNoMewDir(t *testing.T) {
	proj, ac := preflightCtx(t, map[string]string{
		"package.json": `{"name":"p","version":"1.0.0","dependencies":{"lodash":"^4.17.21"}}`,
		// Berry lock (has __metadata) plus a PnP linker declaration.
		"yarn.lock": "__metadata:\n  version: 8\n  cacheKey: 10c0\n\n" +
			"\"lodash@npm:^4.17.21\":\n  version: 4.17.21\n  resolution: \"lodash@npm:4.17.21\"\n",
		".yarnrc.yml": "nodeLinker: pnp\n",
		".pnp.cjs":    "#!/usr/bin/env node\n",
	})
	if _, err := Install(context.Background(), ac, InstallOptions{}); err == nil {
		t.Fatal("expected Yarn PnP install to be rejected")
	}
	assertNoMewDir(t, proj)
}

// A bun.lockb (unsupported binary lock) must be rejected without creating .mew.
func TestPreflightBunLockbRejectedNoMewDir(t *testing.T) {
	proj, ac := preflightCtx(t, map[string]string{
		"package.json": `{"name":"p","version":"1.0.0","dependencies":{"ms":"2.1.3"}}`,
		"bun.lockb":    "\x00\x00binary",
	})
	if _, err := Install(context.Background(), ac, InstallOptions{}); err == nil {
		t.Fatal("expected bun.lockb install to be rejected")
	}
	assertNoMewDir(t, proj)
}

// A workspace filter without the workspaces gate must be rejected without
// creating .mew.
func TestPreflightWorkspaceFilterGateNoMewDir(t *testing.T) {
	proj, ac := preflightCtx(t, map[string]string{
		"package.json": `{"name":"p","version":"1.0.0","private":true,"workspaces":["packages/*"]}`,
	})
	if _, err := Install(context.Background(), ac, InstallOptions{Filter: []string{"pkg-a"}}); err == nil {
		t.Fatal("expected filtered install to be rejected without the workspaces gate")
	}
	assertNoMewDir(t, proj)
}

// installPreflight itself must never touch the filesystem: calling it directly
// on a clean project must leave no .mew and no node_modules.
func TestInstallPreflightIsReadOnly(t *testing.T) {
	proj, ac := preflightCtx(t, map[string]string{
		"package.json": `{"name":"p","version":"1.0.0","dependencies":{"ms":"2.1.3"}}`,
	})
	before := dirEntryNames(t, proj)

	pj, err := project.Open(context.Background(), proj)
	if err != nil {
		t.Fatal(err)
	}
	// CleanNodeModules must not run during preflight even when requested.
	if err := installPreflight(context.Background(), ac, pj, InstallOptions{CleanNodeModules: true}); err != nil {
		t.Fatalf("preflight on a valid project should pass: %v", err)
	}

	assertNoMewDir(t, proj)
	after := dirEntryNames(t, proj)
	if len(before) != len(after) {
		t.Fatalf("preflight changed the project tree:\nbefore=%v\nafter=%v", before, after)
	}
}

func dirEntryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}
