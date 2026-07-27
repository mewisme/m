//go:build windows

package transaction

import (
	"os"
	"syscall"
	"unsafe"
)

func inodeVisitKey(info os.FileInfo) (string, bool) {
	return "", false
}

func createJunction(link, target string) error {
	return junctionDir(target, link)
}

func junctionDir(target, link string) error {
	targetPtr, err := syscall.UTF16PtrFromString(`\\?\` + target)
	if err != nil {
		return err
	}
	linkPtr, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	mod := syscall.NewLazyDLL("kernel32.dll")
	proc := mod.NewProc("CreateSymbolicLinkW")
	const symbolicLinkFlagDirectory = 0x1
	r, _, e := proc.Call(
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
