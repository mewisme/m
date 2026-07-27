package snapshot_test

import (
	"testing"

	"github.com/mewisme/m/internal/snapshot"
)

func TestStoreCreateListPrune(t *testing.T) {
	root := t.TempDir()
	store := snapshot.NewStore(root)
	id, err := store.NextID()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(id, []byte(`{"name":"a"}`), []byte("lock"), "sha256:abc"); err != nil {
		t.Fatal(err)
	}
	id2, err := store.NextID()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(id2, []byte(`{"name":"b"}`), []byte("lock2"), "sha256:def"); err != nil {
		t.Fatal(err)
	}
	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != id2 {
		t.Fatalf("list=%v", list)
	}
	if err := store.Prune(1); err != nil {
		t.Fatal(err)
	}
	list, err = store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != id2 {
		t.Fatalf("after prune: %v", list)
	}
}
