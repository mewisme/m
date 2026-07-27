//go:build !windows && !linux && !darwin

package transaction

import (
	"fmt"
	"syscall"
)

func processStartTime(pid int) (int64, error) {
	return 0, fmt.Errorf("transaction.lock: process start time unsupported on this platform")
}

func currentProcessIdentity() (pid int, start int64, err error) {
	return 0, 0, fmt.Errorf("transaction.lock: process identity unsupported on this platform")
}

func processIdentityAlive(pid int, start int64) bool {
	return false
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
