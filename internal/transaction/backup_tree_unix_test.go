//go:build !windows

package transaction_test

import "syscall"

func syscallMkfifo(path string, mode uint32) error {
	return syscall.Mkfifo(path, mode)
}
