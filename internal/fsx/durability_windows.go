//go:build windows

package fsx

import (
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

func syncFileHandle(f *os.File) error {
	if err := f.Sync(); err != nil {
		if ignorableWindowsDirSyncErr(err) {
			return nil
		}
		return err
	}
	return nil
}

// ignorableWindowsDirSyncErr reports directory fsync failures Windows may return for
// directories the process cannot open for metadata sync. PublishFileDurable treats
// these as best-effort parent sync; callers requiring strict durability must verify
// platform support separately.
func ignorableWindowsDirSyncErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "access is denied")
}

func syncDirHandle(f *os.File) error {
	if err := f.Sync(); err != nil {
		if ignorableWindowsDirSyncErr(err) {
			return nil
		}
		return err
	}
	return nil
}

func replaceExistingFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	srcPtr, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	dstPtr, err := windows.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	if err := windows.MoveFileEx(srcPtr, dstPtr, windows.MOVEFILE_REPLACE_EXISTING); err != nil {
		return err
	}
	return nil
}
