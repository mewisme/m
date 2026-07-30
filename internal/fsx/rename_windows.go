//go:build windows

package fsx

import (
	"context"
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func renamePath(ctx context.Context, src, dst string) error {
	_ = ctx
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !os.IsExist(err) {
		if _, statErr := os.Stat(dst); statErr != nil {
			return err
		}
	}
	srcPtr, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	dstPtr, err := windows.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	flags := uint32(windows.MOVEFILE_REPLACE_EXISTING | windows.MOVEFILE_WRITE_THROUGH)
	if err := windows.MoveFileEx(srcPtr, dstPtr, flags); err != nil {
		return err
	}
	return nil
}

func isTransientRenameErr(err error) bool {
	if err == nil {
		return false
	}
	var errno windows.Errno
	if errors.As(err, &errno) {
		switch errno {
		case windows.ERROR_ACCESS_DENIED, windows.ERROR_SHARING_VIOLATION:
			return true
		}
	}
	// os.PathError from os.Rename may wrap errno differently.
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && pathErr.Err != nil {
		if errno, ok := pathErr.Err.(windows.Errno); ok {
			switch errno {
			case windows.ERROR_ACCESS_DENIED, windows.ERROR_SHARING_VIOLATION:
				return true
			}
		}
	}
	return false
}
