package dlx

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConsentStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mxDir := filepath.Join(dir, "mxcache")
	cacheRoot := dir
	path := ConsentStorePath(cacheRoot)
	key := NewConsentKey(ResolvedEnvironmentIdentity{GraphDigest: "g"}, "cmd", "owner")
	store, err := LoadConsentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.HasConsent(key) {
		t.Fatal("unexpected consent")
	}
	if err := MergeConsent(context.Background(), cacheRoot, mxDir, key); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadConsentStore(path)
	if err != nil || !reloaded.HasConsent(key) {
		t.Fatalf("reload=%+v err=%v", reloaded, err)
	}
}

func TestPublishRequestIndexAtomic(t *testing.T) {
	dir := t.TempDir()
	path := RequestIndexPath(dir, "req")
	doc := RequestIndex{RequestDigest: "req", ResolvedEnvironmentDigest: "env"}
	if err := PublishRequestIndex(path, doc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
