package snapshot_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/mewisme/m/internal/snapshot"
)

func TestSnapshotRoundTrip(t *testing.T) {
	s := &snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion,
		ID:            "snap-1",
		CreatedAt:     time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		GraphDigest:   "sha256:abc",
		PolicyDigest:  "sha256:def",
	}
	first, err := snapshot.EncodeJSON(s)
	if err != nil {
		t.Fatal(err)
	}
	again, err := snapshot.DecodeJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := snapshot.EncodeJSON(again)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("unstable\n%s\n%s", first, second)
	}
}
