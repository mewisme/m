//go:build !linux

package linker

// reflinkFile is implemented per-platform; default tries hardlink.
func reflinkFile(src, dest string) error {
	return applyHardlinkFile(src, dest, 0)
}
