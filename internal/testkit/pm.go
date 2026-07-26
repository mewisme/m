package testkit

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// LookPM returns the path to a package manager binary, or empty if absent.
func LookPM(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

// RunPMResult is the outcome of a reference PM invocation.
type RunPMResult struct {
	Path     string
	Args     []string
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// RunPM runs name with args in dir when the binary exists.
func RunPM(ctx context.Context, name string, args []string, dir string, env []string) RunPMResult {
	path := LookPM(name)
	res := RunPMResult{Path: path, Args: args}
	if path == "" {
		res.Err = exec.ErrNotFound
		res.ExitCode = 127
		return res
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()
	res.Err = err
	if err == nil {
		res.ExitCode = 0
	} else if ee, ok := err.(*exec.ExitError); ok {
		res.ExitCode = ee.ExitCode()
	} else {
		res.ExitCode = 1
	}
	return res
}
