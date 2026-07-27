//go:build windows

package fsx

import (
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

func syncFileHandle(f *os.File) error {
	return f.Sync()
}

func syncDirHandle(f *os.File) error {
	if err := f.Sync(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "access is denied") {
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
