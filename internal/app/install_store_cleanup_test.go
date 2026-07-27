package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/testkit"
	"github.com/mewisme/m/internal/transaction"
)

func testStoreInstallContext(t *testing.T) (*Context, string, func()) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	home := t.TempDir()
	storeDir := filepath.Join(home, "store")
	cacheDir := filepath.Join(home, "cache")
	proj := filepath.Join(home, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{
  "name": "store-cleanup",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD: proj,
		CLI: map[string]any{
			"registry":              srv.URL,
			"link.use_global_store": true,
			"store.dir":             storeDir,
			"cache.dir":             cacheDir,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Context{CWD: proj, Config: eff}, proj, srv.Close
}

func TestFetchAndImportGraphSurfacesStoreCleanup(t *testing.T) {
	ac, _, cleanup := testStoreInstallContext(t)
	defer cleanup()

	store.SetImportLockReleaseTestHook(func(lockDir string) error {
		return apperr.New(apperr.Store, "store.import.lock.release", lockDir, "lock not released: not owner")
	})
	t.Cleanup(func() { store.SetImportLockReleaseTestHook(nil) })

	eng, err := resolver.NewFromApp(ac.Config, nil, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	res, err := eng.Resolve(context.Background(), ac.CWD, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}

	out, err := fetchAndImportGraph(context.Background(), ac, res.Graph)
	if err != nil {
		t.Fatal(err)
	}
	if !out.StoreMaintenanceRequired {
		t.Fatal("expected store maintenance required")
	}
	if len(out.CleanupWarningCodes) == 0 {
		t.Fatal("expected cleanup codes")
	}
	if out.CleanupWarningCodes[0] != store.CleanupCodeImportLockRelease {
		t.Fatalf("code=%q", out.CleanupWarningCodes[0])
	}
}

func TestInstallPreservesStoreWarningsOnLinkFailure(t *testing.T) {
	ac, proj, cleanup := testStoreInstallContext(t)
	defer cleanup()

	store.SetImportLockReleaseTestHook(func(lockDir string) error {
		return apperr.New(apperr.Store, "store.import.lock.release", lockDir, "lock not released: not owner")
	})
	t.Cleanup(func() { store.SetImportLockReleaseTestHook(nil) })

	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, proj)
	if err != nil {
		t.Fatal(err)
	}
	txn := sess.Runner()

	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "post_link" {
			return apperr.New(apperr.Install, "app.install.link", "", "injected link failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	installRes, err := runInstallInSession(ctx, sess, InstallOptions{}, nil, nil)
	if err == nil {
		t.Fatal("expected link failure")
	}
	abortRes, abortErr := abortMutation(ctx, sess, txn, err)
	if abortErr == nil {
		t.Fatal("expected abort error")
	}
	installRes = mergeInstallResults(installRes, abortRes)

	if !installRes.StoreMaintenanceRequired {
		t.Fatal("expected store maintenance flag")
	}
	if installRes.RecoveryRequired {
		t.Fatal("store-only cleanup must not set RecoveryRequired")
	}
	found := false
	for _, c := range installRes.CleanupWarningCodes {
		if c == cleanupCodeStoreImportLockRelease {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing store code in %v", installRes.CleanupWarningCodes)
	}
	summary := FormatInstallSummary(installRes)
	if strings.Contains(summary, "m recover") {
		t.Fatalf("should not suggest recover for store-only cleanup: %q", summary)
	}
	if !strings.Contains(summary, "m store status") {
		t.Fatalf("missing store status hint: %q", summary)
	}
}

func TestMergeStoreCleanupDedupesPairs(t *testing.T) {
	var res InstallResult
	mergeStoreCleanupIntoResult(&res, []string{cleanupCodeStoreImportLockRelease}, []string{"a"})
	mergeStoreCleanupIntoResult(&res, []string{cleanupCodeStoreImportLockRelease}, []string{"a"})
	if len(res.CleanupWarningCodes) != 1 {
		t.Fatalf("codes=%v", res.CleanupWarningCodes)
	}
}

func TestFormatInstallSummaryStoreOnlyCommitted(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:                true,
		StoreMaintenanceRequired: true,
		CleanupWarningCodes:      []string{cleanupCodeStoreImportLockRelease},
		CleanupWarnings:          []string{"lock not released"},
	})
	if strings.Contains(summary, "m recover") {
		t.Fatalf("unexpected recover hint: %q", summary)
	}
	if !strings.Contains(summary, "m store status") {
		t.Fatalf("missing store hint: %q", summary)
	}
}

func TestApplyFetchOutcomeSetsFlags(t *testing.T) {
	var res InstallResult
	applyFetchOutcome(&res, FetchOutcome{
		CleanupWarningCodes:      []string{cleanupCodeStoreIndexLockRelease},
		CleanupWarnings:          []string{"index lock"},
		StoreCleanupIncomplete:   true,
		StoreMaintenanceRequired: true,
	})
	if !res.StoreMaintenanceRequired || !res.StoreCleanupIncomplete {
		t.Fatalf("flags not set: %+v", res)
	}
}
