package fsx

import (
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
)

// PublishFile replaces path with data using platform-accurate publication semantics.
//
// Unix (Linux, macOS): write a temp file in the destination directory, fsync, then rename
// over path. The prior generation remains readable until rename succeeds.
//
// Windows: same temp+rename choreography; os.Rename replaces an existing file without a
// prior delete so a crash after write but before rename leaves the old file intact.
// Generation-scanned heads (journal.head, current.head) recover from numbered siblings.
func PublishFile(path string, data []byte, perm os.FileMode) error {
	return WriteAtomic(path, data, perm)
}

// PublishRename moves src to dst with platform-accurate replacement semantics.
//
// Files: rename into place without deleting dst first (Windows-safe).
// Directories: rename choreography via a .mew-old aside buffer (see publishDir).
func PublishRename(src, dst string) error {
	if isPublishDir(src) || isPublishDir(dst) {
		return publishDir(src, dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", dst, err)
	}
	if err := os.Rename(src, dst); err != nil {
		_ = os.Remove(dst)
		if err2 := os.Rename(src, dst); err2 != nil {
			return apperr.Wrap(apperr.IO, "fsx.publish", dst, err2)
		}
	}
	return nil
}

func isPublishDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func publishDir(stageDir, liveDir string) error {
	backup := liveDir + ".mew-old"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(liveDir); err == nil {
		if err := os.Rename(liveDir, backup); err != nil {
			return apperr.Wrap(apperr.IO, "fsx.publish", liveDir, err)
		}
	}
	if err := os.Rename(stageDir, liveDir); err != nil {
		if _, statErr := os.Stat(backup); statErr == nil {
			_ = os.Rename(backup, liveDir)
		}
		return apperr.Wrap(apperr.IO, "fsx.publish", stageDir, err)
	}
	_ = os.RemoveAll(backup)
	return nil
}
