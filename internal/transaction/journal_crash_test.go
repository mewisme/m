package transaction

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJournalGenerationSurvivesTruncatedHead(t *testing.T) {
	root := t.TempDir()
	runner := NewRunner(root)
	ctx := context.Background()
	if err := runner.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := runner.SetState(StateValidated); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runner.Root, journalHeadName), []byte(`{"generation":99,"checksum":"deadbeef"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadJournalGeneration(runner.Root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.State != StateValidated {
		t.Fatalf("expected validated state from fallback generation, got %+v", loaded)
	}
	_ = runner.Discard(StandaloneFinishOpts())
}

func TestCurrentGenerationSurvivesBadHead(t *testing.T) {
	root := t.TempDir()
	if err := writeCurrentGeneration(root, "txn-a"); err != nil {
		t.Fatal(err)
	}
	if err := writeCurrentGeneration(root, "txn-b"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(TxnRoot(root), currentHeadName), []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	id, err := readCurrentGeneration(root)
	if err != nil {
		t.Fatal(err)
	}
	if id != "txn-b" {
		t.Fatalf("got %q", id)
	}
}

func TestJournalGenerationMonotonicAndRecoverable(t *testing.T) {
	root := t.TempDir()
	runner := NewRunner(root)
	ctx := context.Background()
	if err := runner.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := runner.SetState(StateValidated); err != nil {
		t.Fatal(err)
	}
	gen1 := filepath.Join(runner.Root, "journal.000001.json")
	gen2 := filepath.Join(runner.Root, "journal.000002.json")
	if _, err := os.Stat(gen1); err != nil {
		t.Fatalf("missing gen1: %v", err)
	}
	if _, err := os.Stat(gen2); err != nil {
		t.Fatalf("missing gen2: %v", err)
	}
	head, err := readGenerationHead(filepath.Join(runner.Root, journalHeadName))
	if err != nil {
		t.Fatal(err)
	}
	if head.Generation != 2 {
		t.Fatalf("head generation=%d", head.Generation)
	}
	data, err := os.ReadFile(gen2)
	if err != nil {
		t.Fatal(err)
	}
	if checksumHex(data) != head.Checksum {
		t.Fatal("head checksum mismatch")
	}
	_ = runner.Discard(StandaloneFinishOpts())
}

func TestJournalIncompleteHeadFallsBackToPriorGeneration(t *testing.T) {
	root := t.TempDir()
	runner := NewRunner(root)
	ctx := context.Background()
	if err := runner.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := runner.SetState(StateValidated); err != nil {
		t.Fatal(err)
	}
	gen2Path := filepath.Join(runner.Root, "journal.000002.json")
	gen2Data, err := os.ReadFile(gen2Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(runner.Root, journalHeadName)); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadJournalGeneration(runner.Root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.State != StateValidated {
		t.Fatalf("got %+v", loaded)
	}
	if err := os.WriteFile(gen2Path, []byte("truncated"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err = loadJournalGeneration(runner.Root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.State != StateStaging {
		t.Fatalf("expected staging from gen1 fallback, got %+v", loaded)
	}
	_ = gen2Data
	_ = runner.Discard(StandaloneFinishOpts())
}

func TestJournalPhaseStartedWithoutFilesystemPublish(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "target.txt")
	if err := os.WriteFile(live, []byte("prior"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(root)
	ctx := context.Background()
	if err := runner.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	stage := runner.StagePath()
	staged := filepath.Join(stage, "target.txt")
	if err := os.WriteFile(staged, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := []Op{{Kind: OpRename, Path: "target.txt", Backup: "stage/target.txt"}}
	if err := runner.SetPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := runner.RecordBackup("target.txt"); err != nil {
		t.Fatal(err)
	}
	runner.doc.Plan[0].Phase = PhasePublishStarted
	runner.doc.Plan[0].Progress = ProgressApplying
	if err := runner.saveJournal(); err != nil {
		t.Fatal(err)
	}
	prior, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if string(prior) != "prior" {
		t.Fatalf("live mutated before publish: %q", prior)
	}
	if _, err := runner.Rollback(ctx, StandaloneFinishOpts()); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != "prior" {
		t.Fatalf("rollback should restore prior from backup, got %q", after)
	}
	_ = runner.Finish(false, StandaloneFinishOpts())
}
