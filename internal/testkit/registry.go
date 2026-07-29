package testkit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// RegistryManifestSchemaVersion versions fixture registry manifests.
const RegistryManifestSchemaVersion = 1

// BlobMeta is one checksummed blob in a fixture registry.
type BlobMeta struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// RegistryManifest indexes checksums for all registry blobs.
type RegistryManifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	Blobs         []BlobMeta `json:"blobs"`
}

// FixtureRegistry is a hermetic local npm-shaped registry rooted at fixtures/registry/v1.
type FixtureRegistry struct {
	Root     string
	Manifest RegistryManifest
	blobs    map[string]string // relative path → absolute path
	server   *httptest.Server

	// Test hooks
	ForceStatus int32  // if >0, return this status for packument GETs
	ETag        string // etag value for packuments (default fixed)
	hitCount    atomic.Int64

	publishMu sync.Mutex
	publishes []PublishRecord
}

// PublishRecord captures one registry PUT publish.
type PublishRecord struct {
	Method string
	Path   string
	Auth   string
	OTP    string
	Body   []byte
}

// LoadRegistry loads fixtures/<rel> (default registry/v1), verifies checksums, and returns a registry.
func LoadRegistry(t testing.TB, rel string) *FixtureRegistry {
	t.Helper()
	if rel == "" {
		rel = "registry/v1"
	}
	root := FixtureDir(t, rel)
	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatalf("registry manifest: %v", err)
	}
	var man RegistryManifest
	if err := json.Unmarshal(raw, &man); err != nil {
		t.Fatalf("registry manifest parse: %v", err)
	}
	if man.SchemaVersion != RegistryManifestSchemaVersion {
		t.Fatalf("unsupported registry manifest schemaVersion %d", man.SchemaVersion)
	}
	blobs := make(map[string]string, len(man.Blobs))
	for _, b := range man.Blobs {
		relPath := filepath.FromSlash(b.Path)
		abs := filepath.Join(root, relPath)
		sum, err := fileSHA256(abs)
		if err != nil {
			t.Fatalf("checksum %s: %v", b.Path, err)
		}
		if !strings.EqualFold(sum, b.SHA256) {
			t.Fatalf("checksum mismatch for %s: got %s want %s", b.Path, sum, b.SHA256)
		}
		blobs[filepath.ToSlash(b.Path)] = abs
	}
	return &FixtureRegistry{Root: root, Manifest: man, blobs: blobs, ETag: `"fixture-v1"`}
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Start serves packuments and tarballs over HTTP. Caller cleanup via t.
func (r *FixtureRegistry) Start(t testing.TB) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut {
			r.handlePublish(w, req)
			return
		}
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p := path.Clean("/" + strings.TrimPrefix(req.URL.Path, "/"))
		p = strings.TrimPrefix(p, "/")
		if p == "" || p == "." {
			http.NotFound(w, req)
			return
		}
		if raw, err := url.PathUnescape(p); err == nil {
			p = raw
		}
		if strings.Contains(p, "/-/") {
			parts := strings.Split(p, "/-/")
			if len(parts) != 2 {
				http.NotFound(w, req)
				return
			}
			file := parts[1]
			key := "tarballs/" + file
			abs, ok := r.blobs[key]
			if !ok {
				http.NotFound(w, req)
				return
			}
			http.ServeFile(w, req, abs)
			return
		}
		if st := atomic.LoadInt32(&r.ForceStatus); st > 0 {
			http.Error(w, "forced", int(st))
			return
		}
		r.hitCount.Add(1)
		key := "packuments/" + p + ".json"
		abs, ok := r.blobs[key]
		if !ok {
			http.NotFound(w, req)
			return
		}
		etag := r.ETag
		if etag == "" {
			etag = `"fixture-v1"`
		}
		if match := req.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", etag)
		http.ServeFile(w, req, abs)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	r.server = srv
	return srv
}

// HitCount returns packument GET attempts (including 304).
func (r *FixtureRegistry) HitCount() int64 { return r.hitCount.Load() }

// PublishCount returns recorded PUT publishes.
func (r *FixtureRegistry) PublishCount() int {
	r.publishMu.Lock()
	defer r.publishMu.Unlock()
	return len(r.publishes)
}

// Publishes returns a copy of recorded PUT publishes.
func (r *FixtureRegistry) Publishes() []PublishRecord {
	r.publishMu.Lock()
	defer r.publishMu.Unlock()
	out := make([]PublishRecord, len(r.publishes))
	copy(out, r.publishes)
	return out
}

func (r *FixtureRegistry) handlePublish(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(io.LimitReader(req.Body, 32<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	rec := PublishRecord{
		Method: req.Method,
		Path:   req.URL.Path,
		Auth:   req.Header.Get("Authorization"),
		OTP:    req.Header.Get("npm-otp"),
		Body:   append([]byte(nil), body...),
	}
	r.publishMu.Lock()
	r.publishes = append(r.publishes, rec)
	r.publishMu.Unlock()
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// URL returns the base URL of the started server.
func (r *FixtureRegistry) URL() string {
	if r.server == nil {
		return ""
	}
	return r.server.URL
}

// BlobSHA256 returns the expected sha256 for a relative blob path.
func (r *FixtureRegistry) BlobSHA256(rel string) (string, error) {
	rel = filepath.ToSlash(rel)
	for _, b := range r.Manifest.Blobs {
		if b.Path == rel {
			return b.SHA256, nil
		}
	}
	return "", fmt.Errorf("blob %q not in manifest", rel)
}
