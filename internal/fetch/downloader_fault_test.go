package fetch_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/store"
)

func TestDownloaderRetriesNetworkThenSucceeds(t *testing.T) {
	body := []byte("hello")
	integrity := "sha256-" + hexSHA256(body)
	var calls int
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write(body)
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "cut", http.StatusBadGateway)
			return
		}
		inner.ServeHTTP(w, r)
	}))
	defer srv.Close()
	st := store.NewDir(filepath.Join(t.TempDir(), "blobs"))
	dl := &fetch.Downloader{Client: srv.Client(), Store: st, StagingDir: t.TempDir()}
	_, err := dl.Download(context.Background(), fetch.DownloadRequest{URL: srv.URL, Integrity: integrity})
	if err != nil {
		t.Fatal(err)
	}
	if calls < 2 {
		t.Fatalf("calls=%d want retry", calls)
	}
}

func TestDownloaderNoRetryOnHashMismatch(t *testing.T) {
	body := []byte("hello")
	integrity := "sha256-" + hexSHA256(body)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte("wrong"))
	}))
	defer srv.Close()
	st := store.NewDir(filepath.Join(t.TempDir(), "blobs"))
	dl := &fetch.Downloader{Client: srv.Client(), Store: st, StagingDir: t.TempDir()}
	_, err := dl.Download(context.Background(), fetch.DownloadRequest{URL: srv.URL, Integrity: integrity})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls=%d want no retry on hash mismatch", calls)
	}
}

func TestDownloaderPreferOfflineUsesCache(t *testing.T) {
	body := []byte("cached")
	integrity := "sha256-" + hexSHA256(body)
	st := store.NewDir(filepath.Join(t.TempDir(), "blobs"))
	key := store.Key("sha256/" + hexSHA256(body))
	if err := st.Put(context.Background(), key, body); err != nil {
		t.Fatal(err)
	}
	dl := &fetch.Downloader{
		Client:        &http.Client{},
		Store:         st,
		PreferOffline: true,
	}
	art, err := dl.Download(context.Background(), fetch.DownloadRequest{
		URL: "http://should-not-be-called.invalid/x", Integrity: integrity,
	})
	if err != nil {
		t.Fatal(err)
	}
	if art.Size != int64(len(body)) {
		t.Fatalf("%+v", art)
	}
}

func hexSHA256(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
