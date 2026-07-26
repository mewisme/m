package fetch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fetch"
	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/testkit"
)

func TestDownloaderFixtureTarball(t *testing.T) {
	testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	var integrity string
	for _, b := range reg.Manifest.Blobs {
		if b.Path == "tarballs/lodash-4.17.21.tgz" {
			integrity = "sha256-" + b.SHA256
			break
		}
	}
	if integrity == "" {
		t.Fatal("lodash tarball missing from registry manifest")
	}
	url := srv.URL + "/lodash/-/lodash-4.17.21.tgz"

	cache := filepath.Join(t.TempDir(), "blobs")
	st := store.NewDir(cache)
	dl := &fetch.Downloader{
		Client:     srv.Client(),
		Store:      st,
		StagingDir: t.TempDir(),
	}
	art, err := dl.Download(context.Background(), fetch.DownloadRequest{
		URL: url, Integrity: integrity,
	})
	if err != nil {
		t.Fatal(err)
	}
	if art.Size <= 0 {
		t.Fatalf("%+v", art)
	}
	if !st.Exists(art.BlobKey) {
		t.Fatal("blob missing")
	}
}

func TestDownloaderOfflineHitAndMiss(t *testing.T) {
	testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	blob := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	cache := filepath.Join(t.TempDir(), "blobs")
	st := store.NewDir(cache)
	dl := &fetch.Downloader{Client: srv.Client(), Store: st, StagingDir: t.TempDir()}
	req := fetch.DownloadRequest{URL: srv.URL + "/lodash/-/lodash-4.17.21.tgz", Integrity: blob}
	if _, err := dl.Download(context.Background(), req); err != nil {
		t.Fatal(err)
	}

	offline := &fetch.Downloader{Client: srv.Client(), Store: st, Offline: true}
	if _, err := offline.Download(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	miss := &fetch.Downloader{Client: srv.Client(), Store: st, Offline: true}
	_, err := miss.Download(context.Background(), fetch.DownloadRequest{
		URL: req.URL, Integrity: "sha256-0000000000000000000000000000000000000000000000000000000000000000",
	})
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("got %v", err)
	}
}

func TestDownloaderWorkerCap(t *testing.T) {
	var active atomic.Int32
	var peak atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := active.Add(1)
		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}
		defer active.Add(-1)
		w.WriteHeader(200)
		// body hashes to sha256 e3b0c442...
	}))
	defer srv.Close()

	cache := filepath.Join(t.TempDir(), "blobs")
	st := store.NewDir(cache)
	dl := &fetch.Downloader{Client: srv.Client(), Store: st, StagingDir: t.TempDir(), Workers: 2}
	reqs := make([]fetch.DownloadRequest, 6)
	for i := range reqs {
		reqs[i] = fetch.DownloadRequest{
			URL: srv.URL, Integrity: "sha256-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}
	}
	if _, err := dl.DownloadAll(context.Background(), reqs); err != nil {
		t.Fatal(err)
	}
	if peak.Load() > 2 {
		t.Fatalf("peak concurrency %d want <=2", peak.Load())
	}
}

func TestDownloaderCancelRemovesTemp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()
	staging := t.TempDir()
	st := store.NewDir(filepath.Join(t.TempDir(), "blobs"))
	dl := &fetch.Downloader{Client: srv.Client(), Store: st, StagingDir: staging}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := dl.Download(ctx, fetch.DownloadRequest{
		URL: srv.URL, Integrity: "sha256-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	})
	if err == nil {
		t.Fatal("expected cancel error")
	}
	entries, _ := os.ReadDir(staging)
	for _, e := range entries {
		if stringsHasPrefix(e.Name(), "mew-fetch-") {
			t.Fatalf("left temp %s", e.Name())
		}
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
