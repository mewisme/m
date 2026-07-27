//go:build !windows

package fsx

import "os"

func syncFileHandle(f *os.File) error {
	return f.Sync()
}

func syncDirHandle(f *os.File) error {
	return f.Sync()
}

func replaceExistingFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if _, statErr := os.Stat(dst); statErr != nil {
		return os.Rename(src, dst)
	}
	old := dst + mewOldSuffix
	if err := os.Rename(dst, old); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		if restoreErr := os.Rename(old, dst); restoreErr != nil {
			return restoreErr
		}
		return err
	}
	if err := os.Remove(old); err != nil {
		return err
	}
	return nil
}
