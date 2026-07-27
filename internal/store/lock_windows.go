//go:build windows

package store

import "syscall"

const (
	importLockStillActive                    = 259
	importLockProcessQueryLimitedInformation = 0x1000
)

func importLockProcessAlive(pid int) bool {
	h, err := syscall.OpenProcess(importLockProcessQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == importLockStillActive
}
