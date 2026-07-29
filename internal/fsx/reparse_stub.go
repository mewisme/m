//go:build !windows

package fsx

import "errors"

const IOReparseTagMountPoint = 0
const IOReparseTagSymlink = 0

// ReparseTag reports no reparse tag on non-Windows platforms.
func ReparseTag(path string) uint32 {
	_ = path
	return 0
}

// ReadMountPoint is only supported on Windows.
func ReadMountPoint(path string) (substitute, print string, tag uint32, err error) {
	_ = path
	return "", "", 0, errors.New("fsx: mount points require windows")
}

// CreateMountPoint is only supported on Windows.
func CreateMountPoint(link, substitute, print string) error {
	_, _, _ = link, substitute, print
	return errors.New("fsx: mount points require windows")
}

// ReadSymlinkTarget is only supported on Windows.
func ReadSymlinkTarget(path string) (string, error) {
	_ = path
	return "", errors.New("fsx: symlink reparse points require windows")
}
