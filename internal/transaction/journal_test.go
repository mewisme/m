package transaction_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/transaction"
)

func TestJournalRoundTripGolden(t *testing.T) {
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            "abc123",
		ProjectRoot:   "/proj",
		State:         transaction.StateCommitted,
		Plan: []transaction.Op{
			{Kind: transaction.OpRename, Path: "node_modules", Backup: "stage/node_modules", Progress: transaction.ProgressApplied, Phase: transaction.PhaseApplied},
		},
		Ops: []transaction.Op{
			{Kind: transaction.OpBackup, Path: "m.lock", Backup: "backups/m.lock", Progress: transaction.ProgressApplied,
				DestKind: transaction.DestKindFile, HadPrior: true, PriorKind: transaction.DestKindFile},
		},
	}
	first, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("..", "..", "testdata", "transaction", "journal-samples", "committed.json")
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytesTrimNL(first), bytesTrimNL(want)) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", first, want)
	}
	again, err := transaction.Decode(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := transaction.Encode(again)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("unstable encode\n%s\n%s", first, second)
	}
}

func bytesTrimNL(b []byte) []byte {
	return bytes.TrimSpace(b)
}
