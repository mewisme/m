package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/transaction"
)

func TestTransactionRecoverInterruptedCommit(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "txn-recover",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	lockBefore, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	// Simulate interrupted commit: journal in committing state with backup recorded.
	txnRoot := transaction.TxnRoot(projDir)
	id := "simulated01"
	dir := filepath.Join(txnRoot, id)
	if err := os.MkdirAll(filepath.Join(dir, "backups"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "backups", "m.lock"), lockBefore, 0o644); err != nil {
		t.Fatal(err)
	}
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            id,
		ProjectRoot:   projDir,
		State:         transaction.StateCommitting,
		Ops: []transaction.Op{
			{Kind: transaction.OpBackup, Path: "m.lock", Backup: "backups/m.lock"},
		},
	}
	data, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, transaction.JournalNameV1), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transaction.CurrentPath(projDir), []byte(id+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Corrupt live lock as if partial commit happened.
	if err := os.WriteFile(filepath.Join(projDir, "m.lock"), []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "recover")
	if code != 0 {
		t.Fatalf("recover exit=%d out=%s", code, out)
	}
	after, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, lockBefore) {
		t.Fatalf("lock not restored: %q vs %q", after, lockBefore)
	}
}

func hasDirectDep(t *testing.T, projDir, name string) bool {
	t.Helper()
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
	_, ok := doc.Dependencies[name]
	return ok
}

func TestSnapshotRestoreAfterAdd(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "snap-restore",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	if !strings.Contains(captureM(t, projDir, cfgPath, "snapshot", "list"), "000001") {
		t.Fatal("missing snapshot 000001")
	}
	if code, out := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatalf("add exit=%d out=%s", code, out)
	}
	if !hasDirectDep(t, projDir, "pkg-c") {
		t.Fatal("expected direct pkg-c dependency after add")
	}
	if code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "000001"); code != 0 {
		t.Fatalf("restore exit=%d out=%s", code, out)
	}
	if hasDirectDep(t, projDir, "pkg-c") {
		t.Fatal("direct pkg-c dependency should be removed after restore")
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "pkg-a", "package.json")); err != nil {
		t.Fatal("pkg-a should remain")
	}
}

func TestRollbackCommand(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "rollback",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	if code, _ := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatal("add failed")
	}
	if code, out := runM(t, projDir, cfgPath, "rollback"); code != 0 {
		t.Fatalf("rollback exit=%d out=%s", code, out)
	}
	if hasDirectDep(t, projDir, "pkg-c") {
		t.Fatal("direct pkg-c dependency should be removed after rollback")
	}
}

func TestFailedAddPreservesManifestAndLock(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "fail-add",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	pkgBefore, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	code, _ := runM(t, projDir, cfgPath, "add", "nonexistent-pkg-xyz@1.0.0")
	if code == 0 {
		t.Fatal("expected add failure")
	}
	pkgAfter, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(pkgAfter) != string(pkgBefore) {
		t.Fatal("package.json changed after failed add")
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("m.lock changed after failed add")
	}
}

func TestInstallJournalFlagKeepsTxnDir(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "journal-flag",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, out := runM(t, projDir, cfgPath, "install", "--journal")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	entries, err := os.ReadDir(transaction.TxnRoot(projDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected txn dir retained with --journal")
	}
}

func captureM(t *testing.T, projDir, cfgPath string, args ...string) string {
	t.Helper()
	_, out := runM(t, projDir, cfgPath, args...)
	return out
}

func TestTransactionMidCommitHook(t *testing.T) {
	projDir := t.TempDir()
	testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Setenv("NO_PROXY", "*")
	pkgJSON := `{"name":"hook","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "commit" && opIndex == 0 {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })
	ac, err := app.New(context.Background(), app.Options{CWD: projDir, ConfigPath: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.Install(context.Background(), ac, app.InstallOptions{})
	if err == nil {
		t.Fatal("expected install failure")
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules")); !os.IsNotExist(err) {
		t.Fatal("node_modules should not exist after failed first install commit")
	}
}
