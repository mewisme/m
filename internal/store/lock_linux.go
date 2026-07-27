//go:build linux

package store

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func processStartTime(pid int) (int64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, err
	}
	s := string(data)
	idx := strings.LastIndex(s, ") ")
	if idx < 0 {
		return 0, fmt.Errorf("store.import.lock: malformed /proc stat")
	}
	fields := strings.Fields(s[idx+2:])
	if len(fields) < 20 {
		return 0, fmt.Errorf("store.import.lock: short /proc stat")
	}
	return strconv.ParseInt(fields[19], 10, 64)
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
	return syscall.Kill(pid, 0) == nil
}
