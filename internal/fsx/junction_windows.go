//go:build windows

package fsx

// IsJunction reports whether path is a directory junction (reparse mount point).
func IsJunction(path string) bool {
	return ReparseTag(path) == IOReparseTagMountPoint
}
