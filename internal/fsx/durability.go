package fsx

import (
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
)

const mewOldSuffix = ".mew-old"

var (
	durabilityTestHook func(phase string, path string) error
	syncDirTestHook    func(path string) error
)

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
	if syncDirTestHook != nil {
		if err := syncDirTestHook(path); err != nil {
			return err
		}
	}
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
// Prefer PublishNewFileExclusive or PublishFileDurable for critical paths.
func PublishNewFile(path string, data []byte, perm os.FileMode) error {
	return WriteAtomic(path, data, perm)
}

// PublishNewFileExclusive is PublishFileDurable for paths that must not exist beforehand.
func PublishNewFileExclusive(path string, data []byte, perm os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return apperr.New(apperr.Integrity, "fsx.publish", path, "path already exists")
	} else if !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
	}
	return PublishFileDurable(path, data, perm)
}

// PublishFileDurable writes via temp+sync, rename, SyncFile(dest), and SyncDir(parent).
func PublishFileDurable(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if perm != 0 {
		if err := tmp.Chmod(perm); err != nil {
			_ = tmp.Close()
			return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
		}
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
	}
	if durabilityTestHook != nil {
		if err := durabilityTestHook("pre_rename", path); err != nil {
			return err
		}
	}
	if err := replaceExistingFile(tmpName, path); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
	}
	if durabilityTestHook != nil {
		if err := durabilityTestHook("post_rename", path); err != nil {
			return err
		}
	}
	if err := SyncFile(path); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", path, err)
	}
	if durabilityTestHook != nil {
		if err := durabilityTestHook("post_file_sync", path); err != nil {
			return err
		}
	}
	if err := syncDirForPublish(dir); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", dir, err)
	}
	if durabilityTestHook != nil {
		if err := durabilityTestHook("post_dir_sync", dir); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceFileRecoverable moves src over dst, then syncs dst and its parent directory.
func ReplaceFileRecoverable(src, dst string) error {
	if err := ReplaceExistingFile(src, dst); err != nil {
		return err
	}
	if err := SyncFile(dst); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", dst, err)
	}
	if err := syncDirForPublish(filepath.Dir(dst)); err != nil {
		return apperr.Wrap(apperr.IO, "fsx.publish", dst, err)
	}
	return nil
}

func syncDirForPublish(dir string) error {
	err := SyncDir(dir)
	if err != nil && ignorableWindowsDirSyncErr(err) {
		return nil
	}
	return err
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

// SetDurabilityTestHook installs a hook for PublishFileDurable phase tests.
func SetDurabilityTestHook(fn func(phase string, path string) error) {
	durabilityTestHook = fn
}

// SetSyncDirTestHook installs a hook for SyncDir failure injection in tests.
func SetSyncDirTestHook(fn func(path string) error) {
	syncDirTestHook = fn
}
