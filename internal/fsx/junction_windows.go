//go:build windows

package fsx

import (
	"os"
	"syscall"
	"unsafe"
)

const fileAttributeReparsePoint = 0x00000400

var (
	modKernel32Junction       = syscall.NewLazyDLL("kernel32.dll")
	procGetFileAttributesJunc = modKernel32Junction.NewProc("GetFileAttributesW")
)

// IsJunction reports whether path is a directory junction (reparse mount point).
func IsJunction(path string) bool {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attrs, _, _ := procGetFileAttributesJunc.Call(uintptr(unsafe.Pointer(ptr)))
	if attrs == uintptr(syscall.INVALID_FILE_ATTRIBUTES) {
		return false
	}
	if attrs&fileAttributeReparsePoint == 0 {
		return false
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0 && fi.IsDir()
}
