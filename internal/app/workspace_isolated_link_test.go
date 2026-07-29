package app_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
)

func moduleRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if filepath.Base(path) == "metadata.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceMutationAddIsolatedLink(t *testing.T) {
	testkit.CleanEnv(t)
	testkit.EnableWorkspaces(t)
	t.Setenv("MEW_RESOLVE_AUTO_INSTALL_PEERS", "1")

	src := filepath.Join(moduleRoot(t), "fixtures", "locks", "generated", "pnpm-9", "workspace")
	proj := t.TempDir()
	copyTree(t, src, proj)

	pkgPath := filepath.Join(proj, "package.json")
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["packageManager"] = "pnpm@9.0.0"
	deps, _ := doc["dependencies"].(map[string]any)
	if deps == nil {
		deps = map[string]any{}
	}
	deps["ms"] = "2.1.3"
	doc["dependencies"] = deps
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkgPath, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	loadOpts := config.LoadOptions{CWD: proj, ProjectRoot: proj}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	ac := &app.Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}

	_, err = app.Install(context.Background(), ac, app.InstallOptions{PnpmMajor: 9})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
}
