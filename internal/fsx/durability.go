package fsx

import (
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
)

const mewOldSuffix = ".mew-old"

// SyncFile flushes file contents to stable storage.
// Linux/macOS: fsync on an open read handle. Windows: FlushFileBuffers when available.
func SyncFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "fsx.sync", path, err)
	}
	defer func() { _ = f.Close() }()
	if err := syncFileHandle(f); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.sync", path, err)
	}
	return nil
}

// SyncDir flushes directory metadata so a child rename is durable.
// Call after publishing a file or tree into dir.
func SyncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "fsx.sync", path, err)
	}
	defer func() { _ = f.Close() }()
	if err := syncDirHandle(f); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.sync", path, err)
	}
	return nil
}

// PublishNewFile writes path atomically without deleting a prior generation first.
func PublishNewFile(path string, data []byte, perm os.FileMode) error {
	return WriteAtomic(path, data, perm)
}

// ReplaceExistingFile moves src over dst without a delete-first data-loss window.
func ReplaceExistingFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", dst, err)
	}
	if err := replaceExistingFile(src, dst); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", dst, err)
	}
	return nil
}

// PublishDirectory replaces liveDir with stageDir using journaled .mew-old aside space.
func PublishDirectory(stageDir, liveDir string) error {
	if err := publishDirectory(stageDir, liveDir); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", liveDir, err)
	}
	return SyncDir(filepath.Dir(liveDir))
}

// RecoverDirectoryPublication restores liveDir from .mew-old when publish was interrupted.
func RecoverDirectoryPublication(liveDir string) error {
	backup := liveDir + mewOldSuffix
	liveStat, liveErr := os.Stat(liveDir)
	backupStat, backupErr := os.Stat(backup)
	if liveErr != nil && os.IsNotExist(liveErr) {
		if backupErr == nil && backupStat.IsDir() {
			if err := os.Rename(backup, liveDir); err != nil {
				return apperr.Wrap(apperr.IO, "fsx.publish.recover", liveDir, err)
			}
			return nil
		}
		return nil
	}
	if backupErr == nil && backupStat.IsDir() {
		if liveErr == nil && liveStat.IsDir() {
			if err := os.RemoveAll(backup); err != nil {
				return apperr.Wrap(apperr.IO, "fsx.publish.recover", backup, err)
			}
			return nil
		}
	}
	return nil
}

func publishDirectory(stageDir, liveDir string) error {
	backup := liveDir + mewOldSuffix
	if err := os.RemoveAll(backup); err != nil {
		return err
	}
	if _, err := os.Stat(liveDir); err == nil {
		if err := os.Rename(liveDir, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(stageDir, liveDir); err != nil {
		if _, statErr := os.Stat(backup); statErr == nil {
			if restoreErr := os.Rename(backup, liveDir); restoreErr != nil {
				return restoreErr
			}
		}
		return err
	}
	if err := os.RemoveAll(backup); err != nil {
		return err
	}
	return nil
}

func isPublishDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}
