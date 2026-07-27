package fsx

import (
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
)

// PublishFile replaces path with durable publication semantics.
func PublishFile(path string, data []byte, perm os.FileMode) error {
	return PublishFileDurable(path, data, perm)
}

// PublishRename moves src to dst with platform-accurate replacement semantics.
func PublishRename(src, dst string) error {
	if isPublishDir(src) || isPublishDir(dst) {
		return PublishDirectory(src, dst)
	}
	return ReplaceExistingFile(src, dst)
}

// WriteGenerationExclusive creates path with O_EXCL; rejects duplicate generations.
func WriteGenerationExclusive(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.generation", path, err)
	}
	if perm == 0 {
		perm = 0o644
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		if os.IsExist(err) {
			return apperr.New(apperr.Integrity, "fsx.generation", path, "duplicate generation")
		}
		return apperr.Wrap(apperr.IO, "fsx.generation", path, err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return apperr.Wrap(apperr.IO, "fsx.generation", path, err)
	}
	if err := syncFileHandle(f); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return apperr.Wrap(apperr.IO, "fsx.generation", path, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return apperr.Wrap(apperr.IO, "fsx.generation", path, err)
	}
	return nil
}
