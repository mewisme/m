// Package process supervises child processes, signals, and shells.
package process

import (
	"context"
	"io"
)

// Spec describes a child process to start.
type Spec struct {
	Path   string
	Args   []string
	Dir    string
	Env    []string
	Shell  string // optional Windows shell; ComSpec resolved from Env when empty
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Handle is an opaque running process handle.
type Handle struct {
	PID int
	raw any
}

// ProcessSupervisor starts and waits on child processes.
type ProcessSupervisor interface {
	Start(ctx context.Context, spec Spec) (*Handle, error)
	Wait(ctx context.Context, h *Handle) error
}
