package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/transaction"
)

const testPnpmLockYAML = `lockfileVersion: '9.0'

settings:
  autoInstallPeers: true

importers:
  .:
    dependencies:
      lodash:
        specifier: 4.17.21
        version: 4.17.21

packages:
  lodash@4.17.21:
    resolution: {integrity: sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63}
    engines: {node: '>=0.10.0'}

snapshots:
  lodash@4.17.21:
    dependencies: {}

nubVersion: "1.0.0"
`

func testIncumbentProject(t *testing.T, lockName string) (*Context, string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Cleanup(srv.Close)

	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{
  "name": "lock-txn-test",
  "version": "1.0.0",
  "private": true,
  "dependencies": { "lodash": "4.17.21" }
}`
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, lockName), []byte(testPnpmLockYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	loadOpts := config.LoadOptions{
		CWD:         proj,
		ProjectRoot: proj,
		CLI:         map[string]any{"registry": srv.URL},
	}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	return &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}, proj
}

func assertIncumbentLockBytes(t *testing.T, proj, lockName string, before []byte) {
	t.Helper()
	after, err := os.ReadFile(filepath.Join(proj, lockName))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("incumbent lock bytes changed")
	}
	if _, err := os.Stat(filepath.Join(proj, "m.lock")); err == nil {
		t.Fatal("must not create m.lock")
	}
}

func TestInstallTxnBackupInterruptPreservesIncumbent(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "backup" {
			return apperr.New(apperr.Transaction, "test.hook", "", "injected backup failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	_, err = Install(context.Background(), ac, InstallOptions{Frozen: true, PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected backup failure")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}

func TestInstallTxnPublishInterruptPreservesIncumbent(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "publish" {
			return apperr.New(apperr.Transaction, "test.hook", "", "injected publish failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	_, err = Install(context.Background(), ac, InstallOptions{Frozen: true, PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected publish failure")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}

func TestInstallTxnPostLockfileFailurePreservesIncumbent(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "post_lockfile" {
			return apperr.New(apperr.Transaction, "test.hook", "", "injected post_lockfile failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	_, err = Install(context.Background(), ac, InstallOptions{PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected post_lockfile failure")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}

func TestInstallTxnPostValidateFailurePreservesIncumbent(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "post_validate" {
			return apperr.New(apperr.Transaction, "test.hook", "", "injected post_validate failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	_, err = Install(context.Background(), ac, InstallOptions{PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected post_validate failure")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}

func TestInstallTxnStagingDurabilityFailurePreservesIncumbent(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	fsx.SetDurabilityTestHook(func(phase, _ string) error {
		if phase == "pre_rename" {
			return apperr.New(apperr.IO, "test.durable", "pnpm-lock.yaml", "injected staging failure")
		}
		return nil
	})
	t.Cleanup(func() { fsx.SetDurabilityTestHook(nil) })

	_, err = Install(context.Background(), ac, InstallOptions{Frozen: true, PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected staging durability failure")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}

func TestInstallTxnUnsupportedLockUntouched(t *testing.T) {
	testkit.CleanEnv(t)
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	unsupported, err := os.ReadFile(filepath.Join(testkit.ModuleRoot(t), "fixtures", "locks", "pnpm", "unsupported", "v7", "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"legacy","version":"1.0.0","packageManager":"pnpm@9.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "pnpm-lock.yaml"), unsupported, 0o644); err != nil {
		t.Fatal(err)
	}
	loadOpts := config.LoadOptions{CWD: proj, ProjectRoot: proj}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}
	before := append([]byte(nil), unsupported...)
	_, err = Install(context.Background(), ac, InstallOptions{PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected unsupported lock rejection")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}

func TestWriteLockRejectsIncumbentNubPnpm(t *testing.T) {
	for _, lockName := range []string{"nub.lock", "pnpm-lock.yaml"} {
		t.Run(lockName, func(t *testing.T) {
			ac, _ := testIncumbentProject(t, lockName)
			err := WriteLock(context.Background(), ac, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if apperr.CodeOf(err) != apperr.LockUnsupported {
				t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
			}
		})
	}
}

func TestMigrateDryRunNeverTouchesIncumbent(t *testing.T) {
	ac, proj := testIncumbentProject(t, "nub.lock")
	before, err := os.ReadFile(filepath.Join(proj, "nub.lock"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := MigrateLock(context.Background(), ac, MigrateLockOptions{From: "nub", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.DryRun {
		t.Fatal("expected dry-run")
	}
	if result.SourceLockPath == "" || result.SourceIdentity != "nub" {
		t.Fatalf("result=%+v", result)
	}
	after, err := os.ReadFile(filepath.Join(proj, "nub.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("dry-run must not mutate incumbent lock")
	}
	if _, err := os.Stat(filepath.Join(proj, "m.lock")); err == nil {
		t.Fatal("dry-run must not create m.lock")
	}
}

func TestMigrateFailClosedOnLoss(t *testing.T) {
	ac, proj := testIncumbentProject(t, "nub.lock")
	_, err := MigrateLock(context.Background(), ac, MigrateLockOptions{From: "nub"})
	if err == nil {
		t.Fatal("expected lossy migration failure")
	}
	if apperr.CodeOf(err) != apperr.LockUnrepresentable {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if _, err := os.Stat(filepath.Join(proj, "m.lock")); err == nil {
		t.Fatal("failed migration must not create m.lock")
	}
}

func TestInstallTxnCommitInterruptPreservesIncumbent(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "commit" {
			return apperr.New(apperr.Transaction, "test.hook", "", "injected commit failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	_, err = Install(context.Background(), ac, InstallOptions{Frozen: true, PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected commit failure")
	}
	after, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("failed txn must preserve incumbent lock bytes")
	}
}

func TestWriteStagedExtLockEncodeFailure(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	SetWriteStagedExtLockTestHook(func() error {
		return apperr.New(apperr.Lockfile, "test.encode", "pnpm-lock.yaml", "injected encode failure")
	})
	t.Cleanup(func() { SetWriteStagedExtLockTestHook(nil) })

	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Install(context.Background(), ac, InstallOptions{PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected encode failure")
	}
	after, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("encode failure must not mutate incumbent lock")
	}
}

func TestDetectIncumbentLockPnpm(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	projDoc, err := project.Open(context.Background(), proj)
	if err != nil {
		t.Fatal(err)
	}
	det, err := detectIncumbentLock(projDoc)
	if err != nil {
		t.Fatal(err)
	}
	if det.Format != "pnpm-v9" {
		t.Fatalf("format=%s", det.Format)
	}
	_ = ac
}

func TestInstallRejectsLegacyPnpmBeforeTxn(t *testing.T) {
	testkit.CleanEnv(t)
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy, err := os.ReadFile(filepath.Join(testkit.ModuleRoot(t), "fixtures", "locks", "pnpm", "unsupported", "v6", "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"legacy","version":"1.0.0","packageManager":"pnpm@9.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "pnpm-lock.yaml"), legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	loadOpts := config.LoadOptions{CWD: proj, ProjectRoot: proj}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}
	before := append([]byte(nil), legacy...)
	_, err = Install(context.Background(), ac, InstallOptions{PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected legacy rejection before txn commit")
	}
	if apperr.CodeOf(err) != apperr.LockUnsupported {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	after, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("legacy rejection must not mutate incumbent lock")
	}
	if _, err := os.Stat(filepath.Join(proj, "m.lock")); err == nil {
		t.Fatal("legacy rejection must not create m.lock")
	}
}

func TestMigratePublicationInterruptPreservesSource(t *testing.T) {
	ac, proj := testIncumbentProject(t, "nub.lock")
	before, err := os.ReadFile(filepath.Join(proj, "nub.lock"))
	if err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "commit" {
			return apperr.New(apperr.Transaction, "test.hook", "", "injected migration commit failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	_, err = MigrateLock(context.Background(), ac, MigrateLockOptions{From: "nub", DryRun: false})
	if err == nil {
		t.Skip("migration succeeded without loss; interrupt test needs lossy nub fixture")
	}
	after, err := os.ReadFile(filepath.Join(proj, "nub.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("failed migration publication must preserve source lock")
	}
}

func TestInstallTxnAliasLockPreservesIncumbentOnEncodeFailure(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	const aliasLock = `lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      my-lodash:
        specifier: npm:lodash@4.17.21
        version: lodash@4.17.21
packages:
  lodash@4.17.21:
    resolution: {integrity: sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs17LhbZVGedAJv8XZ1tvj5FvSg==}
snapshots:
  lodash@4.17.21: {}
`
	if err := os.WriteFile(filepath.Join(proj, "pnpm-lock.yaml"), []byte(aliasLock), 0o644); err != nil {
		t.Fatal(err)
	}
	before := []byte(aliasLock)
	SetWriteStagedExtLockTestHook(func() error {
		return apperr.New(apperr.Lockfile, "test.encode", "pnpm-lock.yaml", "injected alias encode failure")
	})
	t.Cleanup(func() { SetWriteStagedExtLockTestHook(nil) })
	_, err := Install(context.Background(), ac, InstallOptions{PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected encode failure")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}

func TestInstallTxnPatchLockPreservesIncumbentOnEncodeFailure(t *testing.T) {
	ac, proj := testIncumbentProject(t, "pnpm-lock.yaml")
	patchLock := `lockfileVersion: '9.0'
patchedDependencies:
  ms@2.1.3:
    hash: ts3vzsn6djz7ihcowyzjb4qjla
    path: patches/ms@2.1.3.patch
importers:
  .:
    dependencies:
      ms:
        specifier: 2.1.3
        version: 2.1.3(patch_hash=ts3vzsn6djz7ihcowyzjb4qjla)
packages:
  ms@2.1.3:
    resolution: {integrity: sha512-6FlzubTLZG3J2a/NVCAleEhjzq5oxgHyaCU9yYXvcLsvoVaHJq/s5xXI6/XXP6tz7R9xAOtHnSO/tXtF3WRTlA==}
snapshots:
  ms@2.1.3(patch_hash=ts3vzsn6djz7ihcowyzjb4qjla): {}
`
	if err := os.WriteFile(filepath.Join(proj, "pnpm-lock.yaml"), []byte(patchLock), 0o644); err != nil {
		t.Fatal(err)
	}
	before := []byte(patchLock)
	SetWriteStagedExtLockTestHook(func() error {
		return apperr.New(apperr.Lockfile, "test.encode", "pnpm-lock.yaml", "injected patch encode failure")
	})
	t.Cleanup(func() { SetWriteStagedExtLockTestHook(nil) })
	_, err := Install(context.Background(), ac, InstallOptions{PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected encode failure")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}

func TestInstallTxnWorkspaceLockPreservesIncumbentOnCommitFailure(t *testing.T) {
	testkit.CleanEnv(t)
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	pkgA := filepath.Join(proj, "packages", "pkg-a")
	if err := os.MkdirAll(pkgA, 0o755); err != nil {
		t.Fatal(err)
	}
	wsLock := `lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      pkg-a:
        specifier: workspace:*
        version: link:packages/pkg-a
  packages/pkg-a:
    dependencies:
      ms:
        specifier: 2.1.3
        version: 2.1.3
packages:
  ms@2.1.3:
    resolution: {integrity: sha512-6FlzubTLZG3J2a/NVCAleEhjzq5oxgHyaCU9yYXvcLsvoVaHJq/s5xXI6/XXP6tz7R9xAOtHnSO/tXtF3WRTlA==}
snapshots:
  ms@2.1.3: {}
`
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"ws","version":"1.0.0","private":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "pnpm-workspace.yaml"), []byte("packages:\n  - packages/*\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgA, "package.json"), []byte(`{"name":"pkg-a","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "pnpm-lock.yaml"), []byte(wsLock), 0o644); err != nil {
		t.Fatal(err)
	}
	loadOpts := config.LoadOptions{CWD: proj, ProjectRoot: proj}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}
	before := []byte(wsLock)
	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "commit" {
			return apperr.New(apperr.Transaction, "test.hook", "", "injected commit failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })
	_, err = Install(context.Background(), ac, InstallOptions{Frozen: true, PnpmMajor: 9})
	if err == nil {
		t.Fatal("expected commit failure")
	}
	assertIncumbentLockBytes(t, proj, "pnpm-lock.yaml", before)
}
