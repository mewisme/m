//go:build windows

package process

import (
	"os"
	"os/exec"
)

func forwardCancelSignal(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	// ponytail: best-effort Windows interrupt before taskkill /F tree-kill.
	_ = cmd.Process.Signal(os.Interrupt)
}
