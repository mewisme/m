//go:build linux

package planner

import (
	"os"
	"syscall"
	"unsafe"
)

func platformReflink(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
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
