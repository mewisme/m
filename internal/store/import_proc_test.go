package store_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/testkit"
)

const importProcEnv = "MEW_STORE_IMPORT_PROC"

func TestImportProcConcurrent(t *testing.T) {
	if role := os.Getenv(importProcEnv); role != "" {
		runImportProcChild(t, role)
		return
	}

	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"

	const workers = 4
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestImportProcConcurrent$", "-test.count=1")
		cmd.Env = append(os.Environ(),
			importProcEnv+"=import",
			"MEW_STORE_ROOT="+root,
			"MEW_STORE_TGZ="+tgz,
			"MEW_STORE_INTEGRITY="+integrity,
		)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		go func(c *exec.Cmd) { errCh <- c.Wait() }(cmd)
	}
	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}

	keys, err := ps.ListPackageKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if err := ps.VerifyPackage(context.Background(), keys[0]); err != nil {
		t.Fatal(err)
	}
}

func runImportProcChild(t *testing.T, role string) {
	t.Helper()
	switch role {
	case "import":
		root := os.Getenv("MEW_STORE_ROOT")
		tgz := os.Getenv("MEW_STORE_TGZ")
		integrity := os.Getenv("MEW_STORE_INTEGRITY")
		if root == "" || tgz == "" || integrity == "" {
			t.Fatal("missing child env")
		}
		ps := store.NewPackageStore(root)
		if _, err := ps.ImportFromTarball(context.Background(), tgz, integrity); err != nil {
			t.Fatal(err)
		}
	case "hold-lock":
		root := os.Getenv("MEW_STORE_ROOT")
		keyHex := os.Getenv("MEW_STORE_KEY_HEX")
		if root == "" || keyHex == "" {
			t.Fatal("missing child env")
		}
		lock := filepath.Join(root, ".locks", "sha256", keyHex+".lock")
		if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
			t.Fatal(err)
		}
		doc, err := json.Marshal(map[string]any{
			"schemaVersion": 1,
			"pid":           os.Getpid(),
			"startedAt":     time.Now().UTC(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(lock, doc, 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3 * time.Second)
		_ = os.Remove(lock)
	case "cancel-wait":
		root := os.Getenv("MEW_STORE_ROOT")
		keyHex := os.Getenv("MEW_STORE_KEY_HEX")
		if root == "" || keyHex == "" {
			t.Fatal("missing child env")
		}
		lock := filepath.Join(root, ".locks", "sha256", keyHex+".lock")
		if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
			t.Fatal(err)
		}
		doc, err := json.Marshal(map[string]any{
			"schemaVersion": 1,
			"pid":           os.Getpid(),
			"startedAt":     time.Now().UTC(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(lock, doc, 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Second)
		_ = os.Remove(lock)
	default:
		t.Fatalf("unknown role %q", role)
	}
}

func TestImportProcStaleLockRecovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := store.PackageKeyFromIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, ".locks", key.Algo, key.Hex+".lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	doc, err := json.Marshal(map[string]any{
		"schemaVersion": 1,
		"pid":           999999999,
		"startedAt":     time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, doc, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ImportFromTarball(context.Background(), tgz, integrity); err != nil {
		t.Fatal(err)
	}
	if store.HasImportLock(ps, key) {
		t.Fatal("import lock should be released")
	}
}

func TestImportProcContextCancel(t *testing.T) {
	if os.Getenv(importProcEnv) == "cancel-wait" {
		runImportProcChild(t, "cancel-wait")
		return
	}

	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := store.PackageKeyFromIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestImportProcContextCancel$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		importProcEnv+"=cancel-wait",
		"MEW_STORE_ROOT="+root,
		"MEW_STORE_KEY_HEX="+key.Hex,
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = ps.ImportFromTarball(ctx, tgz, integrity)
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if apperr.CodeOf(err) != apperr.Cancelled {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	_ = cmd.Wait()
}

func TestImportProcPruneSkipsActiveLock(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := ps.ImportFromTarball(context.Background(), tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, ".locks", key.Algo, key.Hex+".lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	doc, err := json.Marshal(map[string]any{
		"schemaVersion": 1,
		"pid":           os.Getpid(),
		"startedAt":     time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, doc, 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, err := store.PruneCandidates(ps, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no prune candidates with active lock, got %d", len(candidates))
	}
}

func TestImportProcNoQuarantineRace(t *testing.T) {
	if os.Getenv(importProcEnv) == "hold-lock" {
		runImportProcChild(t, "hold-lock")
		return
	}

	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "pkg-cli-1.0.0.tgz")
	integrity := "sha256-6ffb2697417ee0f02ad400c8d92c46cfb5889cf84603cd1f797146fde316b5d0"
	key, err := ps.ImportFromTarball(context.Background(), tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	pkgDir := ps.PackagePath(key)
	if err := os.Chmod(filepath.Join(pkgDir, "package.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestImportProcNoQuarantineRace$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		importProcEnv+"=hold-lock",
		"MEW_STORE_ROOT="+root,
		"MEW_STORE_KEY_HEX="+key.Hex,
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		_, err := ps.ImportFromTarball(context.Background(), tgz, integrity)
		done <- err
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child: %v", err)
	}
	if err := ps.VerifyPackage(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	quarantine := filepath.Join(root, ".quarantine", "sha256", key.Hex)
	if _, err := os.Stat(quarantine); err != nil {
		t.Fatalf("expected quarantine dir: %v", err)
	}
}

func TestImportProcLockPathLayout(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	key := store.PackageKey{Algo: "sha256", Hex: "abc"}
	lock := filepath.Join(root, ".locks", key.Algo, key.Hex+".lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !store.HasImportLock(ps, key) {
		t.Fatal("expected external lock detected")
	}
	if _, err := os.Stat(filepath.Join(root, "packages", "sha256", "abc", ".import.lock")); !os.IsNotExist(err) {
		t.Fatalf("in-package lock should not exist: %v", err)
	}
	_ = lock // lock path exercised by acquire/release above
}
