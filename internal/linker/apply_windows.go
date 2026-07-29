//go:build windows

package linker

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modKernel32            = syscall.NewLazyDLL("kernel32.dll")
	procCreateSymbolicLink = modKernel32.NewProc("CreateSymbolicLinkW")
)

const symbolicLinkFlagDirectory = 0x1

func junctionDir(target, link string) error {
	linkTarget := target
	if rel, err := filepath.Rel(filepath.Dir(link), target); err == nil {
		linkTarget = rel
	}
	targetArg := linkTarget
	if filepath.IsAbs(linkTarget) {
		targetArg = `\\?\` + linkTarget
	}
	targetPtr, err := syscall.UTF16PtrFromString(targetArg)
	if err != nil {
		return err
	}
	linkPtr, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	r, _, e := procCreateSymbolicLink.Call(
		uintptr(unsafe.Pointer(linkPtr)),
		uintptr(unsafe.Pointer(targetPtr)),
		uintptr(symbolicLinkFlagDirectory),
	)
	if r == 0 {
		if e != syscall.Errno(0) {
			return e
		}
		return syscall.EINVAL
	}
	return nil
}
