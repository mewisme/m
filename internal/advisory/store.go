package advisory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

const cacheFileName = "osv.json"

// Store reads and writes the advisory database under <cache>/advisory/osv.json.
type Store struct {
	Dir string
}

// Path returns the on-disk advisory database path.
func (s Store) Path() string {
	return filepath.Join(s.Dir, cacheFileName)
}

// Load reads the cached advisory database.
func (s Store) Load() (*AdvisoryDB, error) {
	data, err := os.ReadFile(s.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.New(apperr.NotFound, "advisory.store", s.Path(),
				"advisory database not found in cache; seed cache/advisory/osv.json")
		}
		return nil, apperr.Wrap(apperr.IO, "advisory.store", s.Path(), err)
	}
	return Load(data)
}

// Write stores raw advisory JSON bytes.
func (s Store) Write(data []byte) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "advisory.store", s.Dir, err)
	}
	path := s.Path()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "advisory.store", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return apperr.Wrap(apperr.IO, "advisory.store", path, err)
	}
	return nil
}

// Refresh downloads advisory JSON from url and writes it to the cache.
func (s Store) Refresh(ctx context.Context, url string, client *http.Client) error {
	if err := ctx.Err(); err != nil {
		return apperr.Wrap(apperr.Cancelled, "advisory.refresh", url, err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "advisory.refresh", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return apperr.Wrap(apperr.Network, "advisory.refresh", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return apperr.New(apperr.Network, "advisory.refresh", url,
			"unexpected status "+resp.Status)
	}
	const maxSize = 64 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return apperr.Wrap(apperr.IO, "advisory.refresh", url, err)
	}
	if len(data) > maxSize {
		return apperr.New(apperr.Integrity, "advisory.refresh", url, "response too large")
	}
	if _, err := Load(data); err != nil {
		return err
	}
	return s.Write(data)
}

// Digest returns the SHA-256 hex digest of advisory bytes.
func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SeedFixture copies testdata/advisory/fixture-osv.json into the store (tests).
func (s Store) SeedFixture(moduleRoot string) error {
	src := filepath.Join(moduleRoot, "testdata", "advisory", "fixture-osv.json")
	data, err := os.ReadFile(src)
	if err != nil {
		return apperr.Wrap(apperr.IO, "advisory.seed", src, err)
	}
	return s.Write(data)
}

// DefaultHTTPClient returns a client with a bounded timeout.
func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Minute}
}
