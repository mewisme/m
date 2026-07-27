package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/snapshot"
)

// RestoreSnapshot copies snapshot manifest+lock to live and reinstalls frozen from cache.
func RestoreSnapshot(ctx context.Context, ac *Context, id string) (InstallResult, error) {
	var res InstallResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	store := snapshot.NewStore(proj.Root)
	rec, err := store.Load(id)
	if err != nil {
		return res, err
	}
	manifestPath := filepath.Join(proj.Root, "package.json")
	lockPath := LockPath(proj.Root)
	if err := writeBytesAtomic(manifestPath, rec.Manifest); err != nil {
		return res, err
	}
	manifest.Invalidate(proj.Root)
	if err := writeBytesAtomic(lockPath, rec.Lock); err != nil {
		return res, err
	}
	return Install(ctx, ac, InstallOptions{Frozen: true})
}

func writeBytesAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return apperr.Wrap(apperr.IO, "app.restore", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "app.restore", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "app.restore", path, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "app.restore", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmpName, path); err2 != nil {
			return apperr.Wrap(apperr.IO, "app.restore", path, err2)
		}
	}
	return nil
}

// Rollback restores the previous snapshot (second-newest).
func Rollback(ctx context.Context, ac *Context) (InstallResult, error) {
	var res InstallResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	list, err := snapshot.NewStore(proj.Root).List()
	if err != nil {
		return res, err
	}
	if len(list) < 2 {
		return res, apperr.New(apperr.NotFound, "app.rollback", "", "no previous snapshot")
	}
	return RestoreSnapshot(ctx, ac, list[1].ID)
}
