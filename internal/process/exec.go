package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// ExecSupervisor is the production ProcessSupervisor using os/exec.
type ExecSupervisor struct{}

// NewExecSupervisor returns a path-restricted process supervisor.
func NewExecSupervisor() *ExecSupervisor {
	return &ExecSupervisor{}
}

type execHandle struct {
	cmd *exec.Cmd
}

// Start launches a child process.
func (s *ExecSupervisor) Start(ctx context.Context, spec Spec) (*Handle, error) {
	if s == nil {
		return nil, errors.New("nil supervisor")
	}
	path, args, err := resolveCommand(spec)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Handle{PID: cmd.Process.Pid, raw: &execHandle{cmd: cmd}}, nil
}

// Wait waits for a started process and returns non-zero exit as error.
func (s *ExecSupervisor) Wait(ctx context.Context, h *Handle) error {
	if h == nil || h.raw == nil {
		return errors.New("invalid handle")
	}
	eh, ok := h.raw.(*execHandle)
	if !ok || eh.cmd == nil {
		return errors.New("invalid handle type")
	}
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- eh.cmd.Wait()
	}()
	select {
	case <-ctx.Done():
		_ = eh.cmd.Process.Kill()
		return ctx.Err()
	case err := <-waitDone:
		if err == nil {
			return nil
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			return &ExitError{Code: code, Err: err}
		}
		return err
	}
}

// ExitError is a non-zero child exit.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("exit status %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ExitCode extracts an exit code from err when available.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) && ee != nil {
		return ee.Code
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr != nil {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
		return exitErr.ExitCode()
	}
	return 1
}

// RestrictedEnv copies base env, prepends binDir to PATH, and strips secrets.
func RestrictedEnv(base []string, binDir string) []string {
	if len(base) == 0 {
		base = os.Environ()
	}
	out := make([]string, 0, len(base))
	seen := map[string]struct{}{}
	for _, kv := range base {
		key := envKey(kv)
		if shouldStripEnv(key) {
			continue
		}
		if strings.EqualFold(key, "PATH") {
			continue
		}
		out = append(out, kv)
		seen[key] = struct{}{}
	}
	pathKey := "PATH"
	if runtime.GOOS == "windows" {
		pathKey = "Path"
	}
	pathVal := binDir
	if old, ok := lookupEnv(base, pathKey); ok && old != "" {
		sep := string(os.PathListSeparator)
		pathVal = binDir + sep + old
	}
	out = append(out, pathKey+"="+pathVal)
	return out
}

func resolveCommand(spec Spec) (string, []string, error) {
	cmd := strings.TrimSpace(spec.Path)
	if cmd == "" {
		return "", nil, errors.New("empty command")
	}
	if len(spec.Args) > 0 {
		return spec.Path, spec.Args, nil
	}
	if runtime.GOOS == "windows" {
		shell := os.Getenv("ComSpec")
		if shell == "" {
			shell = "cmd.exe"
		}
		return shell, []string{"/c", cmd}, nil
	}
	return "sh", []string{"-c", cmd}, nil
}

func envKey(kv string) string {
	if i := strings.IndexByte(kv, '='); i >= 0 {
		return kv[:i]
	}
	return kv
}

func lookupEnv(env []string, key string) (string, bool) {
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			if strings.EqualFold(kv[:i], key) {
				return kv[i+1:], true
			}
		}
	}
	return "", false
}

func shouldStripEnv(key string) bool {
	u := strings.ToUpper(key)
	switch u {
	case "NPM_TOKEN", "NODE_AUTH_TOKEN", "GITHUB_TOKEN", "GITLAB_TOKEN", "NPM_CONFIG__AUTH":
		return true
	}
	if strings.HasPrefix(u, "AWS_") {
		return true
	}
	if strings.Contains(u, "SECRET") || strings.Contains(u, "PASSWORD") {
		return true
	}
	return false
}

// BinDirForPackage returns node_modules/.bin adjacent to packageDir.
func BinDirForPackage(packageDir, nodeModules string) string {
	if nodeModules != "" {
		return filepath.Join(nodeModules, ".bin")
	}
	return filepath.Join(filepath.Dir(packageDir), ".bin")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
