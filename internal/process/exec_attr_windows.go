//go:build windows

package process

import "syscall"

func setProcessGroup(cmdAttr *syscall.SysProcAttr) {}
