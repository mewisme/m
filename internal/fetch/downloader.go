package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/store"
)

const defaultWorkers = 8

// DownloadRequest is one tarball fetch.
type DownloadRequest struct {
	URL       string
	Integrity string
	Shasum    string
	AuthToken string
}

// Artifact is a verified tarball in the blob store.
type Artifact struct {
	Integrity ParsedIntegrity
	BlobKey   store.Key
	BlobPath  string
	Size      int64
}

// Downloader downloads and verifies tarballs into a blob store.
type Downloader struct {
	Client        *http.Client
	Store         *store.Dir
	StagingDir    string
	Workers       int
	Offline       bool
	PreferOffline bool
}

// Download fetches one tarball, verifying integrity before publishing to the store.
func (d *Downloader) Download(ctx context.Context, req DownloadRequest) (*Artifact, error) {
	if d == nil || d.Client == nil || d.Store == nil {
		return nil, apperr.New(apperr.Internal, "fetch.download", redactURL(req.URL), "nil downloader")
	}
	expected, err := ExpectedIntegrity(req.Integrity, req.Shasum)
	if err != nil {
		return nil, err
	}
	key := store.Key(expected.BlobPath())

	if d.Offline || d.PreferOffline {
		if d.Store.Exists(key) {
			return d.artifactFromStore(expected, key)
		}
		if d.Offline {
			return nil, apperr.New(apperr.Network, "fetch.offline", expected.BlobPath(), "blob not in cache")
		}
	}

	data, err := d.downloadWithRetry(ctx, req, expected)
	if err != nil {
		return nil, err
	}
	if err := d.Store.Put(ctx, key, data); err != nil {
		return nil, err
	}
	return &Artifact{
		Integrity: expected,
		BlobKey:   key,
		BlobPath:  d.Store.BlobPath(key),
		Size:      int64(len(data)),
	}, nil
}

// DownloadAll fetches tarballs with bounded concurrency.
func (d *Downloader) DownloadAll(ctx context.Context, reqs []DownloadRequest) ([]*Artifact, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	workers := d.Workers
	if workers <= 0 {
		workers = defaultWorkers
	}
	sem := make(chan struct{}, workers)
	out := make([]*Artifact, len(reqs))
	errs := make([]error, len(reqs))
	var wg sync.WaitGroup
	for i, req := range reqs {
		wg.Add(1)
		go func(i int, req DownloadRequest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			art, err := d.Download(ctx, req)
			out[i] = art
			errs[i] = err
		}(i, req)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (d *Downloader) artifactFromStore(expected ParsedIntegrity, key store.Key) (*Artifact, error) {
	path := d.Store.BlobPath(key)
	st, err := os.Stat(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "fetch.cache", string(key), err)
	}
	return &Artifact{
		Integrity: expected,
		BlobKey:   key,
		BlobPath:  path,
		Size:      st.Size(),
	}, nil
}

func (d *Downloader) downloadWithRetry(ctx context.Context, req DownloadRequest, expected ParsedIntegrity) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := d.downloadOnce(ctx, req, expected)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if apperr.CodeOf(err) == apperr.Integrity {
			return nil, err
		}
	}
	return nil, lastErr
}

func (d *Downloader) downloadOnce(ctx context.Context, req DownloadRequest, expected ParsedIntegrity) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return nil, apperr.Wrap(apperr.Network, "fetch.download", redactURL(req.URL), err)
	}
	if req.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.AuthToken)
	}
	resp, err := d.Client.Do(httpReq)
	if err != nil {
		return nil, apperr.Wrap(apperr.Network, "fetch.download", redactURL(req.URL), err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apperr.New(apperr.Network, "fetch.download", redactURL(req.URL),
			fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	staging := d.StagingDir
	if staging == "" {
		staging = os.TempDir()
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "fetch.staging", staging, err)
	}
	tmp, err := os.CreateTemp(staging, "mew-fetch-*")
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "fetch.staging", staging, err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	h, err := newHasher(expected.Algo)
	if err != nil {
		return nil, err
	}
	limited := &io.LimitedReader{R: resp.Body, N: maxBodyBytes + 1}
	written, err := io.Copy(io.MultiWriter(tmp, h), limited)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "fetch.download", redactURL(req.URL), err)
	}
	if written > maxBodyBytes {
		return nil, apperr.New(apperr.Integrity, "fetch.verify", expected.BlobPath(),
			fmt.Sprintf("body exceeds %d bytes", maxBodyBytes))
	}
	got := fmt.Sprintf("%x", h.Sum(nil))
	if got != expected.Hex {
		return nil, apperr.New(apperr.Integrity, "fetch.verify", expected.BlobPath(),
			fmt.Sprintf("digest mismatch: got %s want %s", got, expected.Hex))
	}
	if err := tmp.Sync(); err != nil {
		return nil, apperr.Wrap(apperr.IO, "fetch.staging", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return nil, apperr.Wrap(apperr.IO, "fetch.staging", tmpPath, err)
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "fetch.staging", tmpPath, err)
	}
	cleanup = false
	_ = os.Remove(tmpPath)
	return data, nil
}
