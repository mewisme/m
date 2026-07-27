package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/transaction"
)

func TestPublishDirOpRecordsParentSyncedInJournal(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "node_modules")
	stage := filepath.Join(root, "stage_nm")
	if err := os.MkdirAll(filepath.Join(live, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(stage, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := transaction.NewRunner(root)
	ctx := context.Background()
	if err := runner.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	plan := []transaction.Op{{Kind: transaction.OpRename, Path: "node_modules", Backup: "stage/node_modules"}}
	if err := os.MkdirAll(filepath.Join(runner.StagePath(), "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runner.SetPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := runner.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}
	if err := runner.Commit(ctx, nil); err != nil {
		t.Fatal(err)
	}
	found := false
	entries, err := os.ReadDir(runner.Root)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		if !strings.HasPrefix(ent.Name(), "journal.") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(runner.Root, ent.Name()))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := transaction.Decode(data)
		if err != nil {
			continue
		}
		for _, op := range doc.Plan {
			if op.Phase == transaction.PhaseParentSynced {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("expected parent_synced phase in journal generation")
	}
	_ = runner.Finish(false)
}
