package transaction_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/transaction"
)

func populateNodeModules(t *testing.T, root string, pkgCount int) {
	t.Helper()
	for i := 0; i < pkgCount; i++ {
		pkg := filepath.Join(root, fmt.Sprintf("pkg-%d", i), "node_modules", "nested")
		if err := os.MkdirAll(pkg, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkg, "index.js"), []byte("module.exports = {};\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func commitInstallPlan(t *testing.T, root string, manifest, lock []byte, stageNM string) {
	t.Helper()
	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	stage := txn.StagePath()
	if err := os.WriteFile(filepath.Join(stage, "package.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "m.lock"), lock, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(stageNM, filepath.Join(stage, "node_modules")); err != nil {
		t.Fatal(err)
	}
	plan := []transaction.Op{
		{Kind: transaction.OpRename, Path: "package.json", Backup: "stage/package.json"},
		{Kind: transaction.OpRename, Path: "m.lock", Backup: "stage/m.lock"},
		{Kind: transaction.OpRename, Path: "node_modules", Backup: "stage/node_modules"},
	}
	if err := txn.SetPlan(plan); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"package.json", "m.lock", "node_modules"} {
		if err := txn.RecordBackup(rel); err != nil {
			t.Fatal(err)
		}
	}
	if err := txn.Commit(ctx, nil); err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	_ = txn.Finish(false, transaction.StandaloneFinishOpts())
}

func TestRepeatedNodeModulesPublish(t *testing.T) {
	root := t.TempDir()
	liveNM := filepath.Join(root, "node_modules")
	populateNodeModules(t, liveNM, 30)

	stageNM1 := filepath.Join(root, "stage-nm-1")
	populateNodeModules(t, stageNM1, 30)
	commitInstallPlan(t, root,
		[]byte(`{"name":"mewjs-test","version":"1.0.0","dependencies":{"axios":"1.0.0"}}`),
		[]byte("lock-v1\n"),
		stageNM1,
	)

	stageNM2 := filepath.Join(root, "stage-nm-2")
	populateNodeModules(t, stageNM2, 31)
	commitInstallPlan(t, root,
		[]byte(`{"name":"mewjs-test","version":"1.0.0","dependencies":{"axios":"1.0.0","lodash":"4.17.21"}}`),
		[]byte("lock-v2\n"),
		stageNM2,
	)

	got, err := os.ReadFile(filepath.Join(liveNM, "pkg-30", "node_modules", "nested", "index.js"))
	if err != nil {
		t.Fatalf("expected published tree from second commit: %v", err)
	}
	if string(got) != "module.exports = {};\n" {
		t.Fatalf("unexpected content: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules.mew-old")); err == nil {
		t.Fatal("stale node_modules.mew-old after successful publish")
	}
}
