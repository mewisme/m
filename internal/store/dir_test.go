package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/store"
)

func TestDirPutGet(t *testing.T) {
	root := filepath.Join(t.TempDir(), "blobs")
	d := store.NewDir(root)
	key := store.Key("sha256/abc")
	content := []byte("verified bytes")
	if err := d.Put(context.Background(), key, content); err != nil {
		t.Fatal(err)
	}
	got, err := d.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("%q", got)
	}
}

func TestDirGetMissing(t *testing.T) {
	d := store.NewDir(t.TempDir())
	_, err := d.Get(context.Background(), "sha256/missing")
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("%v", err)
	}
}

func TestDirPutAtomic(t *testing.T) {
	root := t.TempDir()
	d := store.NewDir(root)
	key := store.Key("sha256/deadbeef")
	if err := d.Put(context.Background(), key, []byte("a")); err != nil {
		t.Fatal(err)
	}
	path := d.BlobPath(key)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
