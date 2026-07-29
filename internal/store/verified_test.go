package store_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/store"
)

func TestPutVerifiedSHA256(t *testing.T) {
	d := store.NewDir(t.TempDir())
	content := []byte("verified tarball bytes")
	sum := sha256.Sum256(content)
	key := store.Key("sha256/" + hex.EncodeToString(sum[:]))
	if err := d.PutVerified(context.Background(), key, bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	got, err := d.GetVerified(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch")
	}
}

func TestPutVerifiedSHA512(t *testing.T) {
	d := store.NewDir(t.TempDir())
	content := []byte("sha512 payload")
	sum := sha512.Sum512(content)
	key := store.Key("sha512/" + hex.EncodeToString(sum[:]))
	if err := d.PutVerified(context.Background(), key, bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	exists, err := d.ExistsVerified(key)
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}

func TestPutVerifiedKeyMismatch(t *testing.T) {
	d := store.NewDir(t.TempDir())
	key := store.Key("sha256/" + strings.Repeat("a", 64))
	err := d.PutVerified(context.Background(), key, bytes.NewReader([]byte("wrong")))
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
}

func TestPutVerifiedMalformedKey(t *testing.T) {
	d := store.NewDir(t.TempDir())
	err := d.PutVerified(context.Background(), "not-a-key", bytes.NewReader(nil))
	if apperr.CodeOf(err) != apperr.Store {
		t.Fatalf("got %v", err)
	}
}

func TestExistsVerifiedCorruptQuarantines(t *testing.T) {
	d := store.NewDir(t.TempDir())
	content := []byte("good")
	sum := sha256.Sum256(content)
	hexDigest := hex.EncodeToString(sum[:])
	key := store.Key("sha256/" + hexDigest)
	path := d.BlobPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	exists, err := d.ExistsVerified(key)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected corrupt blob to be absent")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected blob removed from store path")
	}
	quarantine := filepath.Join(d.Root, ".quarantine", "sha256", hexDigest)
	if _, err := os.Stat(quarantine); err != nil {
		t.Fatalf("expected quarantine: %v", err)
	}
}

func TestGetVerifiedCorrupt(t *testing.T) {
	d := store.NewDir(t.TempDir())
	content := []byte("payload")
	sum := sha256.Sum256(content)
	hexDigest := hex.EncodeToString(sum[:])
	key := store.Key("sha256/" + hexDigest)
	path := d.BlobPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("truncated"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := d.GetVerified(context.Background(), key)
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
}

func TestPutVerifiedConcurrentMatch(t *testing.T) {
	d := store.NewDir(t.TempDir())
	content := []byte("shared immutable blob")
	sum := sha256.Sum256(content)
	key := store.Key("sha256/" + hex.EncodeToString(sum[:]))
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- d.PutVerified(context.Background(), key, bytes.NewReader(content))
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestPutVerifiedConcurrentConflict(t *testing.T) {
	d := store.NewDir(t.TempDir())
	sum := sha256.Sum256([]byte("first"))
	key := store.Key("sha256/" + hex.EncodeToString(sum[:]))
	if err := d.PutVerified(context.Background(), key, bytes.NewReader([]byte("first"))); err != nil {
		t.Fatal(err)
	}
	err := d.PutVerified(context.Background(), key, bytes.NewReader([]byte("second")))
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
}

func TestOpenVerifiedStreaming(t *testing.T) {
	d := store.NewDir(t.TempDir())
	content := []byte("stream me")
	sum := sha256.Sum256(content)
	key := store.Key("sha256/" + hex.EncodeToString(sum[:]))
	if err := d.PutVerified(context.Background(), key, bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	rc, err := d.OpenVerified(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(rc)
	if err != nil {
		_ = rc.Close()
		t.Fatal(err)
	}
	if err := rc.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch")
	}
}

func TestOpenVerifiedCorruptOnClose(t *testing.T) {
	d := store.NewDir(t.TempDir())
	content := []byte("payload")
	sum := sha256.Sum256(content)
	hexDigest := hex.EncodeToString(sum[:])
	key := store.Key("sha256/" + hexDigest)
	path := d.BlobPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("wrong"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc, err := d.OpenVerified(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(rc)
	if err := rc.Close(); apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
}

func TestPutVerifiedWritesCanonicalPath(t *testing.T) {
	root := t.TempDir()
	d := store.NewDir(root)
	content := []byte("complete")
	sum := sha256.Sum256(content)
	key := store.Key("sha256/" + hex.EncodeToString(sum[:]))
	path := d.BlobPath(key)
	if err := d.PutVerified(context.Background(), key, bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	exists, err := d.ExistsVerified(key)
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}
