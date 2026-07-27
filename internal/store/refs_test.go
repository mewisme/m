package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/transaction"
)

func TestCollectReferencedFromTxnJournal(t *testing.T) {
	root := t.TempDir()
	txnDir := filepath.Join(root, ".mew", "txn", "abc123")
	stageDir := filepath.Join(txnDir, "stage", ".mew")
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"schemaVersion": 1,
		"packages":      []string{"sha256-abc"},
	}
	raw, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(stageDir, "store-manifest.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	journal := transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            "abc123",
		ProjectRoot:   root,
		State:         transaction.StateValidated,
	}
	jraw, err := transaction.Encode(&journal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(txnDir, transaction.JournalName), jraw, 0o644); err != nil {
		t.Fatal(err)
	}
	refs, err := store.CollectReferencedIntegrities([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := refs["sha256-abc"]; !ok {
		t.Fatal("expected txn staged manifest reference")
	}
}

func TestCollectReferencedIgnoresCommittedTxn(t *testing.T) {
	root := t.TempDir()
	txnDir := filepath.Join(root, ".mew", "txn", "done")
	stageDir := filepath.Join(txnDir, "stage", ".mew")
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{"schemaVersion": 1, "packages": []string{"sha256-stale"}}
	raw, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(stageDir, "store-manifest.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	journal := transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            "done",
		ProjectRoot:   root,
		State:         transaction.StateCommitted,
	}
	jraw, _ := transaction.Encode(&journal)
	if err := os.WriteFile(filepath.Join(txnDir, transaction.JournalName), jraw, 0o644); err != nil {
		t.Fatal(err)
	}
	refs, err := store.CollectReferencedIntegrities([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := refs["sha256-stale"]; ok {
		t.Fatal("committed txn should not contribute references")
	}
}

func TestPruneCandidatesDeterministic(t *testing.T) {
	root := t.TempDir()
	ps := store.NewPackageStore(root)
	keys := []store.PackageKey{
		{Algo: "sha256", Hex: "bbbb"},
		{Algo: "sha256", Hex: "aaaa"},
	}
	for _, key := range keys {
		dir := ps.PackagePath(key)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	candidates, err := store.PruneCandidates(ps, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates", len(candidates))
	}
	if candidates[0].Hex != "aaaa" || candidates[1].Hex != "bbbb" {
		t.Fatalf("order not deterministic: %v", candidates)
	}
}
