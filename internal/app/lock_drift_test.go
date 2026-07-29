package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker/planner"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/project"
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

func TestManifestDriftsFromLockNoIncumbentLock(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	proj, err := project.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	drift, err := manifestDriftsFromLock(context.Background(), proj)
	if err != nil {
		t.Fatal(err)
	}
	if drift {
		t.Fatal("expected no drift when incumbent lock is absent")
	}
}

func TestManifestDriftsFromLockRemovedRootDep(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "workspace:*" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(`lockfileVersion: "9.0"
importers:
  .:
    dependencies:
      pkg-a:
        specifier: workspace:*
        version: link:packages/pkg-a
      ms:
        specifier: 2.1.2
        version: 2.1.2
packages:
  ms@2.1.2: {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	proj, err := project.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	proj.Identity = project.IdentityPNPM
	norm, err := manifest.ToNormalized(proj.Doc)
	if err != nil {
		t.Fatal(err)
	}
	proj.Normalized = norm
	drift, err := manifestDriftsFromLock(context.Background(), proj)
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift when lock lists removed root dependency")
	}
}

func TestWorkspaceRemoveResolveNoRootMsEdge(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("isolated workspace layout probe runs on Linux CI")
	}
	testkit.CleanEnv(t)
	testkit.EnableWorkspaces(t)
	t.Setenv("MEW_RESOLVE_AUTO_INSTALL_PEERS", "1")

	src := filepath.Join(moduleRoot(t), "fixtures", "locks", "generated", "pnpm-9", "workspace")
	proj := t.TempDir()
	copyTree(t, src, proj)

	writePkg := func(body string) {
		if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePkg(`{
  "name": "fixture-workspace-root",
  "version": "1.0.0",
  "private": true,
  "packageManager": "pnpm@9.0.0",
  "dependencies": { "pkg-a": "workspace:*", "ms": "2.1.3" }
}`)
	loadOpts := config.LoadOptions{CWD: proj, ProjectRoot: proj}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}
	if _, err := Install(context.Background(), ac, InstallOptions{PnpmMajor: 9}); err != nil {
		t.Fatal(err)
	}
	writePkg(`{
  "name": "fixture-workspace-root",
  "version": "1.0.0",
  "private": true,
  "packageManager": "pnpm@9.0.0",
  "dependencies": { "pkg-a": "workspace:*", "ms": "2.1.2" }
}`)
	if _, err := Install(context.Background(), ac, InstallOptions{PnpmMajor: 9}); err != nil {
		t.Fatal(err)
	}
	writePkg(`{
  "name": "fixture-workspace-root",
  "version": "1.0.0",
  "private": true,
  "packageManager": "pnpm@9.0.0",
  "dependencies": { "pkg-a": "workspace:*" }
}`)
	openProj, err := project.Open(context.Background(), proj)
	if err != nil {
		t.Fatal(err)
	}
	drift, err := manifestDriftsFromLock(context.Background(), openProj)
	if err != nil || !drift {
		t.Fatalf("drift=%v err=%v", drift, err)
	}
	sess, err := BeginMutationSession(context.Background(), ac, proj)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(context.Background()) }()
	sessionProj, err := sess.ReopenProject(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	res, err := resolveForInstall(context.Background(), sac, sessionProj, InstallOptions{PnpmMajor: 9}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range res.Graph.Edges {
		if e.From == string(graph.RootImporter) && e.Name == "ms" {
			t.Fatalf("unexpected root ms edge after remove resolve: %v", e)
		}
	}
	stageNM := filepath.Join(t.TempDir(), "node_modules")
	if err := os.MkdirAll(stageNM, 0o755); err != nil {
		t.Fatal(err)
	}
	extractDir := filepath.Join(t.TempDir(), "extract")
	localExtracts, err := buildLocalExtractDirs(proj, res, res.Graph)
	if err != nil {
		t.Fatal(err)
	}
	fetchOut, err := fetchPackages(context.Background(), sac, sessionProj, res.Graph, res.Extensions, extractDir, false, localExtracts)
	if err != nil {
		t.Fatal(err)
	}
	caps, _ := planner.ProbeCached(config.CacheRoot(sac.Config), extractDir, stageNM)
	useStore := config.UseGlobalStore(sac.Config)
	if useStore {
		fetchOut, err = fetchPackages(context.Background(), sac, sessionProj, res.Graph, res.Extensions, extractDir, useStore, localExtracts)
		if err != nil {
			t.Fatal(err)
		}
	}
	linkerMode, err := resolveLinkerMode(context.Background(), sac, sessionProj, InstallOptions{PnpmMajor: 9})
	if err != nil {
		t.Fatal(err)
	}
	lnk := newLinker(linkerMode, linkerOpts{
		NodeModules:  stageNM,
		ExtractDirs:  fetchOut.Extracts,
		Capabilities: caps,
		UseSmartLink: useStore,
	})
	plan, err := lnk.Plan(context.Background(), res.Graph)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range plan.Ops {
		dest := op.Dest
		if strings.Contains(dest, filepath.Join(stageNM, "ms")) && !strings.Contains(dest, ".pnpm") {
			t.Fatalf("isolated plan creates root ms op: %s %s", op.Kind, dest)
		}
	}
}

func TestWorkspaceRemoveInstallDropsRootMs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("isolated workspace layout probe runs on Linux CI")
	}
	testkit.CleanEnv(t)
	testkit.EnableWorkspaces(t)
	t.Setenv("MEW_RESOLVE_AUTO_INSTALL_PEERS", "1")

	src := filepath.Join(moduleRoot(t), "fixtures", "locks", "generated", "pnpm-9", "workspace")
	proj := t.TempDir()
	copyTree(t, src, proj)

	writePkg := func(body string) {
		if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePkg(`{
  "name": "fixture-workspace-root",
  "version": "1.0.0",
  "private": true,
  "packageManager": "pnpm@9.0.0",
  "dependencies": { "pkg-a": "workspace:*", "ms": "2.1.3" }
}`)
	loadOpts := config.LoadOptions{CWD: proj, ProjectRoot: proj}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}
	if _, err := Install(context.Background(), ac, InstallOptions{PnpmMajor: 9}); err != nil {
		t.Fatal(err)
	}
	writePkg(`{
  "name": "fixture-workspace-root",
  "version": "1.0.0",
  "private": true,
  "packageManager": "pnpm@9.0.0",
  "dependencies": { "pkg-a": "workspace:*", "ms": "2.1.2" }
}`)
	if _, err := Install(context.Background(), ac, InstallOptions{PnpmMajor: 9}); err != nil {
		t.Fatal(err)
	}
	writePkg(`{
  "name": "fixture-workspace-root",
  "version": "1.0.0",
  "private": true,
  "packageManager": "pnpm@9.0.0",
  "dependencies": { "pkg-a": "workspace:*" }
}`)
	if _, err := Install(context.Background(), ac, InstallOptions{PnpmMajor: 9}); err != nil {
		t.Fatal(err)
	}
	msRoot := filepath.Join(proj, "node_modules", "ms")
	if _, err := os.Stat(msRoot); err == nil {
		t.Fatalf("root node_modules/ms still present after remove install: %s", msRoot)
	}
}
