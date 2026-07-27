//go:build darwin

package store

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func processStartTime(pid int) (int64, error) {
	k, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return 0, err
	}
	return int64(k.Proc.P_starttime.Sec)*1e9 + int64(k.Proc.P_starttime.Usec)*1e3, nil
}

func currentProcessIdentity() (pid int, start int64, err error) {
	pid = os.Getpid()
	start, err = processStartTime(pid)
	if err != nil {
		return 0, 0, fmt.Errorf("store.import.lock: process identity unavailable: %w", err)
	}
	return pid, start, nil
}

func processIdentityAlive(pid int, start int64) bool {
	if pid <= 0 {
		return false
	}
	actual, err := processStartTime(pid)
	if err != nil {
		return false
	}
	return actual == start
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
