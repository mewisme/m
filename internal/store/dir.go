package store

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
)

// Dir is a minimal content-addressed blob store: <root>/<algo>/<hex>.
type Dir struct {
	Root string
}

// NewDir returns a blob store rooted at dir.
func NewDir(root string) *Dir {
	return &Dir{Root: root}
}

var _ Store = (*Dir)(nil)

// BlobPath returns the on-disk path for an algo/hex key.
func (d *Dir) BlobPath(key Key) string {
	return filepath.Join(d.Root, filepath.FromSlash(string(key)))
}

// Get reads a verified blob by content key (algo/hex).
func (d *Dir) Get(ctx context.Context, key Key) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if d == nil || d.Root == "" {
		return nil, apperr.New(apperr.IO, "store.get", string(key), "nil store")
	}
	path := d.BlobPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.Wrap(apperr.NotFound, "store.get", string(key), err)
		}
		return nil, apperr.Wrap(apperr.IO, "store.get", string(key), err)
	}
	return data, nil
}

// Put atomically writes content to <root>/<key>.
func (d *Dir) Put(ctx context.Context, key Key, content []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.Root == "" {
		return apperr.New(apperr.IO, "store.put", string(key), "nil store")
	}
	pk, err := parseKey(key)
	if err != nil {
		return err
	}
	if err := verifyBytesDigest(content, pk); err != nil {
		return err
	}
	path := d.BlobPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "store.put", string(key), err)
	}
	return writeAtomicVerified(path, pk, content)
}

// Exists reports whether key is present.
func (d *Dir) Exists(key Key) bool {
	if d == nil || d.Root == "" {
		return false
	}
	_, err := os.Stat(d.BlobPath(key))
	return err == nil
}
