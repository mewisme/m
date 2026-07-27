//go:build windows

package fsx_test

import (
	"syscall"
	"testing"
	"unsafe"

	"github.com/mewisme/m/internal/fsx"
)

var (
	modKernel32            = syscall.NewLazyDLL("kernel32.dll")
	procCreateSymbolicLink = modKernel32.NewProc("CreateSymbolicLinkW")
)

const symbolicLinkFlagDirectory = 0x1

func junctionDir(target, link string) error {
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

func TestGuardAncestorsJunctionEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	mew := root + `\proj\.mew`
	if err := junctionDir(outside, mew); err != nil {
		t.Skip("junction not supported:", err)
	}
	if err := fsx.GuardAncestors(root+`\proj`, mew+`\txn`); err == nil {
		t.Fatal("expected junction rejection")
	}
}
