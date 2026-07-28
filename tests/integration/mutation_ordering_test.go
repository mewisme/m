package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/transaction"
)

const mutationOrderingEnv = "MEW_MUTATION_ORDERING_PROC"

func TestConcurrentAddFooBar(t *testing.T) {
	if role := os.Getenv(mutationOrderingEnv); role != "" {
		runMutationOrderingChild(t, role)
		return
	}
	if testing.Short() {
		t.Skip("concurrent add proc test")
	}
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "concurrent-add",
  "version": "1.0.0"
}`)
	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, pkg := range []string{"pkg-a", "pkg-b"} {
		pkg := pkg
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(exe, "-test.run=^TestConcurrentAddFooBar$", "-test.count=1")
			cmd.Env = append(os.Environ(),
				mutationOrderingEnv+"=add",
				"MEW_MUTATION_PROJ="+projDir,
				"MEW_MUTATION_CFG="+cfgPath,
				"MEW_MUTATION_PKG="+pkg,
			)
			if err := cmd.Run(); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(projDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Dependencies["pkg-a"] == "" || doc.Dependencies["pkg-b"] == "" {
		t.Fatalf("expected both deps, got %+v", doc.Dependencies)
	}
}

func runMutationOrderingChild(t *testing.T, role string) {
	t.Helper()
	projDir := os.Getenv("MEW_MUTATION_PROJ")
	cfgPath := os.Getenv("MEW_MUTATION_CFG")
	pkg := os.Getenv("MEW_MUTATION_PKG")
	if projDir == "" || cfgPath == "" {
		t.Fatal("missing project env")
	}
	switch role {
	case "add":
		if pkg == "" {
			t.Fatal("missing MEW_MUTATION_PKG")
		}
		code, out := runM(t, projDir, cfgPath, "add", pkg)
		if code != 0 {
			t.Fatalf("add %s exit=%d out=%s", pkg, code, out)
		}
	default:
		t.Fatalf("unknown role %q", role)
	}
}

func TestInstallRecoversIncompleteBeforeResolve(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "recover-before-resolve",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	txnRoot := transaction.TxnRoot(projDir)
	dir := filepath.Join(txnRoot, "crash")
	if err := os.MkdirAll(filepath.Join(dir, "stage"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            "crash",
		ProjectRoot:   projDir,
		State:         transaction.StateStaging,
	}
	data, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "journal.000001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("retry install exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "pkg-a", "package.json")); err != nil {
		t.Fatal("pkg-a should remain linked after recovery")
	}
	txns, err := transaction.ScanIncompleteTxns(projDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 0 {
		t.Fatalf("expected clean txn state, got %+v", txns)
	}
}

func TestInstallCancelDuringLockWait(t *testing.T) {
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "cancel-wait",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(projDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, projDir, "holder"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.ReleaseProjectLock(projDir, "holder") }()

	ac, err := app.New(context.Background(), app.Options{CWD: projDir, ConfigPath: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = app.Install(waitCtx, ac, app.InstallOptions{})
	if err == nil {
		t.Fatal("expected lock wait cancellation")
	}
	if apperr.CodeOf(err) != apperr.Cancelled && apperr.CodeOf(err) != apperr.Transaction {
		t.Fatalf("unexpected code %s: %v", apperr.CodeOf(err), err)
	}
	after, err := os.ReadFile(filepath.Join(projDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("package.json changed during cancelled lock wait")
	}
	if _, err := os.Stat(filepath.Join(projDir, "m.lock")); !os.IsNotExist(err) {
		t.Fatal("m.lock should not be created during cancelled wait")
	}
}
