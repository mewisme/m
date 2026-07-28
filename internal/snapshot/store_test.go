package snapshot_test

import (
	"os"
	"path/filepath"
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
	if err := store.Create(id, []byte(`{"name":"a"}`), []byte("lock"), "sha256:abc", nil); err != nil {
		t.Fatal(err)
	}
	id2, err := store.NextID()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(id2, []byte(`{"name":"b"}`), []byte("lock2"), "sha256:def", nil); err != nil {
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

func TestStoreV2MemberManifestRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := snapshot.NewStore(root)
	members := map[string][]byte{
		"packages/a/package.json": []byte(`{"name":"a"}` + "\n"),
	}
	if err := store.Create("000001", []byte(`{"name":"root"}`), []byte("lock"), "sha256:abc", members); err != nil {
		t.Fatal(err)
	}
	rec, err := store.Load("000001")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Meta.SchemaVersion != snapshot.SchemaVersion {
		t.Fatalf("schema=%d", rec.Meta.SchemaVersion)
	}
	if len(rec.Meta.MemberManifests) != 1 {
		t.Fatalf("member list=%v", rec.Meta.MemberManifests)
	}
	got := rec.MemberManifests["packages/a/package.json"]
	if string(got) != string(members["packages/a/package.json"]) {
		t.Fatalf("member bytes=%q", got)
	}
}

func TestStoreV1BackwardCompat(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".mew", "snapshots", "000001")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"schemaVersion":1,"id":"000001","createdAt":"2026-01-01T00:00:00Z","graphDigest":"sha256:abc"}`
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"root"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "m.lock"), []byte("lock"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := snapshot.NewStore(root)
	rec, err := store.Load("000001")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Meta.SchemaVersion != snapshot.SchemaVersionV1 {
		t.Fatalf("schema=%d", rec.Meta.SchemaVersion)
	}
	if len(rec.MemberManifests) != 0 {
		t.Fatalf("v1 should have no members: %v", rec.MemberManifests)
	}
}

func TestStoreRejectsTraversalMemberPath(t *testing.T) {
	root := t.TempDir()
	store := snapshot.NewStore(root)
	err := store.Create("000001", []byte(`{"name":"root"}`), []byte("lock"), "sha256:abc", map[string][]byte{
		"../escape/package.json": []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}
