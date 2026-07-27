package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/testkit"
	"github.com/mewisme/m/internal/transaction"
)

func TestUsesStagedSnapshotInputs(t *testing.T) {
	g := &graph.Graph{}
	if usesStagedSnapshotInputs(InstallOptions{}) {
		t.Fatal("empty opts should not use staged snapshot")
	}
	if usesStagedSnapshotInputs(InstallOptions{PreResolvedGraph: g}) {
		t.Fatal("graph alone is insufficient")
	}
	if !usesStagedSnapshotInputs(InstallOptions{PreResolvedGraph: g, StagedManifest: []byte(`{}`)}) {
		t.Fatal("graph + staged manifest should skip live frozen validation")
	}
	if !usesStagedSnapshotInputs(InstallOptions{PreResolvedGraph: g, StagedLock: []byte(`{}`)}) {
		t.Fatal("graph + staged lock should skip live frozen validation")
	}
}

func TestBeginMutationSessionCancelledContext(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac := &Context{
		CWD:    root,
		Config: &config.Effective{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := BeginMutationSession(ctx, ac, root)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !isLockWaitCancellation(err) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if apperr.CodeOf(err) != apperr.Cancelled {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestBeginMutationSessionReopenProject(t *testing.T) {
	root := t.TempDir()
	manifest := `{"name":"mut-session","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	ac := &Context{
		CWD:    root,
		Config: &config.Effective{},
	}
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
	if proj.Doc.Name != "mut-session" {
		t.Fatalf("name=%q", proj.Doc.Name)
	}
	if sess.Project() != proj {
		t.Fatal("Project() should return reopened project")
	}
}

func TestBeginMutationSessionRecoversBeforeReopen(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"recover"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	txnRoot := transaction.TxnRoot(root)
	dir := filepath.Join(txnRoot, "stale")
	if err := os.MkdirAll(filepath.Join(dir, "stage"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            "stale",
		ProjectRoot:   root,
		State:         transaction.StateStaging,
	}
	data, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "journal.000001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	ac := &Context{CWD: root, Config: &config.Effective{}}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	txns, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 1 || txns[0].ID == "stale" {
		t.Fatalf("expected one new incomplete txn, got %+v", txns)
	}
	if _, err := sess.ReopenProject(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestIsLockWaitCancellation(t *testing.T) {
	if !isLockWaitCancellation(context.Canceled) {
		t.Fatal("context.Canceled")
	}
	if !isLockWaitCancellation(apperr.Wrap(apperr.Cancelled, "x", "", context.DeadlineExceeded)) {
		t.Fatal("wrapped deadline")
	}
	if isLockWaitCancellation(errors.New("other")) {
		t.Fatal("unexpected match")
	}
}

func TestAppContextBeforeReopenReturnsError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"no-reopen"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: root, Config: &config.Effective{}}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	_, err = sess.AppContext()
	if err == nil {
		t.Fatal("expected error before ReopenProject")
	}
	if apperr.CodeOf(err) != apperr.Internal {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestMutationSessionReloadEffectiveConfig(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"cfg-reload","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	baseEff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         root,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	loadSpec := config.LoadSpecFromOptions(config.LoadOptions{
		CWD:         root,
		ProjectRoot: root,
	})
	ac := &Context{CWD: root, Config: baseEff, ConfigLoadSpec: loadSpec}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	cfgPath := filepath.Join(root, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"install.linker":"isolated"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := sess.ReopenProject(ctx); err != nil {
		t.Fatal(err)
	}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	if sac == nil || sac.Config == nil {
		t.Fatal("missing session app context")
	}
	if got := config.String(sac.Config, "install.linker", ""); got != "isolated" {
		t.Fatalf("reloaded linker=%q want isolated", got)
	}
	if got := config.String(ac.Config, "install.linker", ""); got != "auto" {
		t.Fatalf("shared context should keep pre-reload default: %q", got)
	}
	if ac.ConfigLoadSpec.ProjectRoot != root {
		t.Fatalf("shared ConfigLoadSpec mutated: %+v", ac.ConfigLoadSpec)
	}
}

func TestMutationSessionReloadPreservesExplicitConfigPath(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"explicit-cfg","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(root, "custom.jsonc")
	if err := os.WriteFile(custom, []byte(`{"install.linker":"hoisted"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: root, ConfigPath: custom})
	if err != nil {
		t.Fatal(err)
	}
	if ac.ConfigLoadSpec.ProjectPath != custom {
		t.Fatalf("stored path=%q want %q", ac.ConfigLoadSpec.ProjectPath, custom)
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	if err := os.WriteFile(custom, []byte(`{"install.linker":"isolated"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := sess.ReloadEffectiveConfig(ctx); err != nil {
		t.Fatal(err)
	}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	if got := config.String(sac.Config, "install.linker", ""); got != "isolated" {
		t.Fatalf("reloaded linker=%q", got)
	}
	opts := ac.ConfigLoadSpec.Clone().WithProjectRoot(root).LoadOptions()
	if opts.ProjectPath != custom {
		t.Fatalf("reload path=%q", opts.ProjectPath)
	}
}

func TestMutationSessionReloadPreservesFrozenGlobalPath(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"global-snap","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(root, "global-cfg")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	globalFile := filepath.Join(cfgDir, "config.jsonc")
	if err := os.WriteFile(globalFile, []byte(`{"offline":true}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"MEW_CONFIG_DIR=" + cfgDir}
	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: root, Env: env})
	if err != nil {
		t.Fatal(err)
	}
	wantGlobal := ac.ConfigLoadSpec.GlobalPath
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	t.Setenv("MEW_CONFIG_DIR", filepath.Join(root, "other"))
	if err := os.WriteFile(globalFile, []byte(`{"offline":false}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := sess.ReloadEffectiveConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if ac.ConfigLoadSpec.GlobalPath != wantGlobal {
		t.Fatalf("global path changed: %q -> %q", wantGlobal, ac.ConfigLoadSpec.GlobalPath)
	}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := config.Get(sac.Config, "offline"); v.Raw != false {
		t.Fatalf("reloaded offline=%v", v.Raw)
	}
}

func TestMutationSessionReloadUsesEnvSnapshot(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"env-snap","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEW_OFFLINE", "true")
	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: root, Env: []string{"MEW_OFFLINE=true"}})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	t.Setenv("MEW_OFFLINE", "false")
	if err := sess.ReloadEffectiveConfig(ctx); err != nil {
		t.Fatal(err)
	}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := config.Get(sac.Config, "offline"); v.Raw != true {
		t.Fatalf("offline=%v", v.Raw)
	}
}

func TestMutationSessionReloadPreservesInitializedEmptyEnv(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"empty-reload","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEW_OFFLINE", "true")
	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: root, Env: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	t.Setenv("MEW_OFFLINE", "false")
	if err := sess.ReloadEffectiveConfig(ctx); err != nil {
		t.Fatal(err)
	}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := config.Get(sac.Config, "offline"); v.Raw != false {
		t.Fatalf("offline=%v want false (empty env must not inherit ambient)", v.Raw)
	}
	if !sac.Config.Env.Initialized() {
		t.Fatal("expected initialized-empty snapshot after reload")
	}
}

func TestMutationSessionReloadMissingExplicitConfig(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"missing-cfg","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(root, "custom.jsonc")
	if err := os.WriteFile(custom, []byte(`{"install.linker":"hoisted"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: root, ConfigPath: custom})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	if err := os.Remove(custom); err != nil {
		t.Fatal(err)
	}
	err = sess.ReloadEffectiveConfig(ctx)
	if err == nil {
		t.Fatal("expected config error")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestMutationSessionScopedRegistryReload(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	defer srv.Close()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"scope-reload","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	baseEff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         root,
		ProjectRoot: root,
		CLI:         map[string]any{"registry": srv.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	loadSpec := config.LoadSpecFromOptions(config.LoadOptions{
		CWD:         root,
		ProjectRoot: root,
		CLI:         map[string]any{"registry": srv.URL},
	})
	ac := &Context{CWD: root, Config: baseEff, ConfigLoadSpec: loadSpec}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	scopeURL := srv.URL + "/scoped"
	if err := os.WriteFile(filepath.Join(root, "m.jsonc"), []byte(`{"registry":"`+srv.URL+`","registries":{"@scope":"`+scopeURL+`"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := sess.ReloadEffectiveConfig(ctx); err != nil {
		t.Fatal(err)
	}
	proj := &project.Project{Root: root, Identity: project.IdentityMew}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	got := registry.ResolveBaseForPackage(sac.Config, root, proj.Identity, "@scope/pkg")
	if got != scopeURL {
		t.Fatalf("scoped registry=%q want %q", got, scopeURL)
	}
	if before := registry.ResolveBaseForPackage(ac.Config, root, proj.Identity, "@scope/pkg"); before == scopeURL {
		t.Fatalf("shared context should not pick up reloaded scope mapping: %q", before)
	}
}

func TestMutationSessionAppContextClonesConfigLoadSpec(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"clone-spec","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"MEW_CACHE_DIR=" + filepath.Join(root, "cache")}
	loadSpec := config.LoadSpecFromOptions(config.LoadOptions{
		CWD:         root,
		ProjectRoot: root,
		Env:         env,
		EnvSnapshot: config.NewEnvSnapshot(env, "linux"),
	})
	baseEff, err := config.Load(context.Background(), loadSpec.LoadOptions())
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: root, Config: baseEff, ConfigLoadSpec: loadSpec}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()
	if _, err := sess.ReopenProject(ctx); err != nil {
		t.Fatal(err)
	}
	sac, err := sess.AppContext()
	if err != nil {
		t.Fatal(err)
	}
	if len(sac.ConfigLoadSpec.Env) == 0 {
		t.Fatal("expected env slice in session spec")
	}
	sac.ConfigLoadSpec.Env[0] = "MEW_CACHE_DIR=/mutated"
	if sac.ConfigLoadSpec.CLI == nil {
		sac.ConfigLoadSpec.CLI = map[string]any{}
	}
	sac.ConfigLoadSpec.CLI["offline"] = true
	if ac.ConfigLoadSpec.Env[0] == "MEW_CACHE_DIR=/mutated" {
		t.Fatal("shared ConfigLoadSpec.Env was mutated")
	}
	if ac.ConfigLoadSpec.CLI != nil && ac.ConfigLoadSpec.CLI["offline"] == true {
		t.Fatal("shared ConfigLoadSpec.CLI was mutated")
	}
}
