//go:build !windows

package process

import (
	"os/exec"
	"syscall"
)

func forwardCancelSignal(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGINT)
	_ = syscall.Kill(-pid, syscall.SIGTERM)
}
