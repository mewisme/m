//go:build !windows

package fsx

// IsJunction reports whether path is a Windows directory junction.
func IsJunction(path string) bool {
	return false
}
