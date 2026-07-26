// Package process supervises child processes, signals, and shells.
package process

import "context"

// Spec describes a child process to start.
type Spec struct {
	Path string
	Args []string
	Dir  string
	Env  []string
}

// Handle is an opaque running process handle.
type Handle struct {
	PID int
}

// ProcessSupervisor starts and waits on child processes.
type ProcessSupervisor interface {
	Start(ctx context.Context, spec Spec) (*Handle, error)
	Wait(ctx context.Context, h *Handle) error
}
