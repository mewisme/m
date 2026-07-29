package fetch

import (
	"context"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
)

// MaterializeLocalTarball extracts a project-relative tarball into dest.
func MaterializeLocalTarball(ctx context.Context, projectRoot, relPath, dest string) error {
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
	if err := archive.Extract(ctx, src, dest, archive.DefaultOptions()); err != nil {
		_ = removeAll(dest)
		return apperr.Wrap(apperr.Integrity, "fetch.tarball", relPath, err)
	}
	return nil
}
