package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
	"github.com/mewisme/m/internal/transaction"
)

func TestWriteOpRecordsFilePublishPhases(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "package.json")
	if err := os.WriteFile(live, []byte(`{"name":"old"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := transaction.NewRunner(root)
	ctx := context.Background()
	if err := runner.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	stageFile := filepath.Join(runner.StagePath(), "package.json")
	if err := os.MkdirAll(filepath.Dir(stageFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stageFile, []byte(`{"name":"new"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := []transaction.Op{{Kind: transaction.OpWrite, Path: "package.json", Backup: "stage/package.json"}}
	if err := runner.SetPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := runner.RecordBackup("package.json"); err != nil {
		t.Fatal(err)
	}
	if err := runner.Commit(ctx, nil); err != nil {
		t.Fatal(err)
	}

	phases := map[string]bool{}
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
			if op.Phase != "" {
				phases[op.Phase] = true
			}
		}
	}
	for _, want := range []string{
		transaction.PhaseWritten,
		transaction.PhaseSynced,
		transaction.PhasePublished,
		transaction.PhaseParentSynced,
	} {
		if !phases[want] {
			t.Fatalf("missing phase %q in journal generations: %#v", want, phases)
		}
	}
	got, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"name":"new"}`+"\n" {
		t.Fatalf("package.json not updated: %q", got)
	}
	_ = runner.Finish(false, transaction.StandaloneFinishOpts())
}

func TestWriteOpParentSyncFailureDoesNotSetParentSynced(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "m.lock")
	if err := os.WriteFile(live, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := transaction.NewRunner(root)
	ctx := context.Background()
	if err := runner.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	stageFile := filepath.Join(runner.StagePath(), "m.lock")
	if err := os.WriteFile(stageFile, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := []transaction.Op{{Kind: transaction.OpWrite, Path: "m.lock", Backup: "stage/m.lock"}}
	if err := runner.SetPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := runner.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}

	fsx.SetSyncDirTestHook(func(dir string) error {
		return apperr.New(apperr.IO, "fsx.sync", dir, "injected parent sync failure")
	})
	t.Cleanup(func() { fsx.SetSyncDirTestHook(nil) })

	err := runner.Commit(ctx, nil)
	if err == nil {
		t.Fatal("expected publish failure")
	}

	phases := map[string]bool{}
	entries, err := os.ReadDir(runner.Root)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		if !strings.HasPrefix(ent.Name(), "journal.") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(runner.Root, ent.Name()))
		if readErr != nil {
			t.Fatal(readErr)
		}
		doc, decErr := transaction.Decode(data)
		if decErr != nil {
			continue
		}
		for _, op := range doc.Plan {
			if op.Phase != "" {
				phases[op.Phase] = true
			}
		}
	}
	if phases[transaction.PhaseParentSynced] {
		t.Fatal("parent_synced should not be recorded when publish fails")
	}
	_ = runner.Finish(false, transaction.StandaloneFinishOpts())
}
