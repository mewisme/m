//go:build windows

package transaction

import (
	"os"
	"syscall"
)

const (
	stillActive                    = 259
	processQueryLimitedInformation = 0x1000
)

func processStartTime(pid int) (int64, error) {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	var creation, exit, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return 0, err
	}
	return int64(creation.HighDateTime)<<32 | int64(creation.LowDateTime), nil
}

func currentProcessIdentity() (pid int, start int64, err error) {
	pid = os.Getpid()
	start, err = processStartTime(pid)
	return pid, start, err
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
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}
