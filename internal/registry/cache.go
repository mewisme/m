package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

// cacheMeta is persisted beside packument.json.
type cacheMeta struct {
	SchemaVersion int       `json:"schemaVersion"`
	ETag          string    `json:"etag,omitempty"`
	FetchedAt     time.Time `json:"fetchedAt"`
	Registry      string    `json:"registry"`
	Name          string    `json:"name"`
	SHA256        string    `json:"sha256"`
}

// DiskCache stores packument bodies under <root>/<originHash>/<escapedName>/.
type DiskCache struct {
	Root string
}

func (c *DiskCache) entryDir(registryOrigin, name string) string {
	sum := sha256.Sum256([]byte(strings.TrimRight(registryOrigin, "/")))
	origin := hex.EncodeToString(sum[:8])
	esc := strings.ReplaceAll(name, "/", "%2F")
	return filepath.Join(c.Root, origin, esc)
}

// Lookup returns cached body and etag when present and valid.
func (c *DiskCache) Lookup(registryOrigin, name string) (body []byte, etag string, ok bool) {
	if c == nil || c.Root == "" {
		return nil, "", false
	}
	dir := c.entryDir(registryOrigin, name)
	metaPath := filepath.Join(dir, "meta.json")
	bodyPath := filepath.Join(dir, "packument.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, "", false
	}
	var meta cacheMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil || meta.SchemaVersion != CacheSchemaVersion {
		_ = c.Evict(registryOrigin, name)
		return nil, "", false
	}
	body, err = os.ReadFile(bodyPath)
	if err != nil {
		_ = c.Evict(registryOrigin, name)
		return nil, "", false
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != meta.SHA256 {
		_ = c.Evict(registryOrigin, name)
		return nil, "", false
	}
	return body, meta.ETag, true
}

// Store writes packument bytes and etag atomically.
func (c *DiskCache) Store(registryOrigin, name, etag string, body []byte) error {
	if c == nil || c.Root == "" {
		return nil
	}
	dir := c.entryDir(registryOrigin, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "registry.cache", dir, err)
	}
	sum := sha256.Sum256(body)
	meta := cacheMeta{
		SchemaVersion: CacheSchemaVersion,
		ETag:          etag,
		FetchedAt:     time.Now().UTC(),
		Registry:      registryOrigin,
		Name:          name,
		SHA256:        hex.EncodeToString(sum[:]),
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "registry.cache", name, err)
	}
	metaBytes = append(metaBytes, '\n')
	bodyPath := filepath.Join(dir, "packument.json")
	metaPath := filepath.Join(dir, "meta.json")
	if err := writeAtomic(bodyPath, body); err != nil {
		return err
	}
	return writeAtomic(metaPath, metaBytes)
}

// Evict removes a cache entry.
func (c *DiskCache) Evict(registryOrigin, name string) error {
	if c == nil || c.Root == "" {
		return nil
	}
	dir := c.entryDir(registryOrigin, name)
	_ = os.RemoveAll(dir)
	return nil
}

// Inspect returns cache paths and etag for CLI.
func (c *DiskCache) Inspect(registryOrigin, name string) (dir, etag string, present bool) {
	body, etag, ok := c.Lookup(registryOrigin, name)
	if !ok {
		return c.entryDir(registryOrigin, name), "", false
	}
	_ = body
	return c.entryDir(registryOrigin, name), etag, true
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return apperr.Wrap(apperr.IO, "registry.cache", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "registry.cache", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "registry.cache", path, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "registry.cache", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmpName, path); err2 != nil {
			return apperr.Wrap(apperr.IO, "registry.cache", path, err2)
		}
	}
	return nil
}
