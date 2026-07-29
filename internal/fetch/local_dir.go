package fetch

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/fsx"
)

// MaterializeLocalDir copies a project-relative directory into dest.
func MaterializeLocalDir(ctx context.Context, projectRoot, relPath, dest string) error {
	src, err := guardedProjectPath(projectRoot, relPath)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := removeAndMkdir(dest); err != nil {
		return err
	}
	return archive.CopyDirTree(src, dest)
}

func guardedProjectPath(projectRoot, relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "", apperr.New(apperr.Usage, "fetch.local", relPath, "empty path")
	}
	rootAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "fetch.local", projectRoot, err)
	}
	target := filepath.Clean(filepath.Join(rootAbs, filepath.FromSlash(relPath)))
	if filepath.IsAbs(relPath) {
		target = filepath.Clean(relPath)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "fetch.local", relPath, err)
	}
	if err := fsx.GuardAncestors(rootAbs, targetAbs); err != nil {
		return "", apperr.Wrap(apperr.Resolve, "fetch.local", relPath, err)
	}
	return targetAbs, nil
}

func removeAndMkdir(dest string) error {
	if err := removeDir(dest); err != nil {
		return err
	}
	return mkdirAll(dest)
}

func removeDir(path string) error {
	if path == "" {
		return nil
	}
	if err := removeAll(path); err != nil {
		return apperr.Wrap(apperr.IO, "fetch.local", path, err)
	}
	return nil
}

func mkdirAll(path string) error {
	if err := mkdir(path, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "fetch.local", path, err)
	}
	return nil
}
