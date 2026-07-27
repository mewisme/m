//go:build windows

package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modKernel32       = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx    = modKernel32.NewProc("LockFileEx")
	procUnlockFileEx  = modKernel32.NewProc("UnlockFileEx")
	lockfileFailImm   = 0x00000001
	lockfileExclusive = 0x00000002
)

type overlapped struct {
	internal     uint64
	internalHigh uint64
	offset       uint32
	offsetHigh   uint32
	hEvent       syscall.Handle
}

func tryAcquireTakeoverGuard(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	var ov overlapped
	r, _, e := procLockFileEx.Call(
		uintptr(f.Fd()),
		uintptr(lockfileExclusive|lockfileFailImm),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&ov)),
	)
	if r == 0 {
		_ = f.Close()
		if e != syscall.Errno(0) {
			return nil, e
		}
		return nil, syscall.Errno(33)
	}
	return func() {
		var unlockOv overlapped
		_, _, _ = procUnlockFileEx.Call(
			uintptr(f.Fd()),
			0,
			1,
			0,
			uintptr(unsafe.Pointer(&unlockOv)),
		)
		_ = f.Close()
	}, nil
}

func isTakeoverGuardBusy(err error) bool {
	return errors.Is(err, syscall.Errno(33))
}
