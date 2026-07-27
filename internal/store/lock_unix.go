//go:build !windows

package store

import "syscall"

func importLockProcessAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
