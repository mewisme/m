//go:build windows

package planner

import (
	"syscall"
	"unsafe"
)

func platformJunction(target, link string) error {
	targetPtr, err := syscall.UTF16PtrFromString(`\\?\` + target)
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

var (
	modKernel32            = syscall.NewLazyDLL("kernel32.dll")
	procCreateSymbolicLink = modKernel32.NewProc("CreateSymbolicLinkW")
)

const symbolicLinkFlagDirectory = 0x1
