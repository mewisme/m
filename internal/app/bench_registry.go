package app

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
)

type fixtureRegistry struct {
	blobs map[string]string
	srv   *httptest.Server
}

func startFixtureRegistry(root string) (*fixtureRegistry, error) {
	manifestPath := filepath.Join(root, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var man struct {
		SchemaVersion int `json:"schemaVersion"`
		Blobs         []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"blobs"`
	}
	if err := json.Unmarshal(raw, &man); err != nil {
		return nil, err
	}
	if man.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported registry manifest schemaVersion %d", man.SchemaVersion)
	}
	blobs := make(map[string]string, len(man.Blobs))
	for _, b := range man.Blobs {
		rel := filepath.FromSlash(b.Path)
		abs := filepath.Join(root, rel)
		sum, err := fileSHA256(abs)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(sum, b.SHA256) {
			return nil, fmt.Errorf("checksum mismatch for %s", b.Path)
		}
		blobs[filepath.ToSlash(b.Path)] = abs
	}
	reg := &fixtureRegistry{blobs: blobs}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
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
			key := "tarballs/" + parts[1]
			abs, ok := blobs[key]
			if !ok {
				http.NotFound(w, req)
				return
			}
			http.ServeFile(w, req, abs)
			return
		}
		key := "packuments/" + p + ".json"
		abs, ok := blobs[key]
		if !ok {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, req, abs)
	})
	reg.srv = httptest.NewServer(mux)
	return reg, nil
}

func (r *fixtureRegistry) URL() string {
	if r == nil || r.srv == nil {
		return ""
	}
	return r.srv.URL
}

func (r *fixtureRegistry) Close() {
	if r != nil && r.srv != nil {
		r.srv.Close()
	}
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
