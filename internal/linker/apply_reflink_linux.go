//go:build linux

package linker

import (
	"os"
	"syscall"
	"unsafe"
)

func reflinkFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	_ = os.Remove(dest)
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, in.Fd(), 0x40049409, uintptr(unsafe.Pointer(&[]uintptr{out.Fd()}[0])))
	if errno != 0 {
		return errno
	}
	return nil
}
