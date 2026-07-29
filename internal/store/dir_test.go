package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/store"
)

func TestDirPutGet(t *testing.T) {
	root := filepath.Join(t.TempDir(), "blobs")
	d := store.NewDir(root)
	content := []byte("verified bytes")
	sum := sha256.Sum256(content)
	key := store.Key("sha256/" + hex.EncodeToString(sum[:]))
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
	content := []byte("a")
	sum := sha256.Sum256(content)
	key := store.Key("sha256/" + hex.EncodeToString(sum[:]))
	if err := d.Put(context.Background(), key, content); err != nil {
		t.Fatal(err)
	}
	path := d.BlobPath(key)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
