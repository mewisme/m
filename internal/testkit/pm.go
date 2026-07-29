package testkit

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
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

// RequirePM returns the path to name or skips/fails the test when absent.
// When MEW_CONFORMANCE_REQUIRE_TOOLS=1, a missing binary is fatal.
func RequirePM(t testing.TB, name string) string {
	t.Helper()
	path := LookPM(name)
	if path == "" {
		msg := name + " not found on PATH"
		if conformanceRequireTools() {
			t.Fatal(msg)
		}
		t.Skip(msg)
	}
	recordConformanceTool(t, name, path)
	return path
}

func conformanceRequireTools() bool {
	return os.Getenv("MEW_CONFORMANCE_REQUIRE_TOOLS") == "1"
}

func recordConformanceTool(t testing.TB, name, path string) {
	t.Helper()
	version := probePMVersion(path)
	entry := name + "=" + path
	if version != "" {
		entry += "@" + version
	}
	key := "MEW_CONFORMANCE_TOOL_RECORD"
	existing := os.Getenv(key)
	if existing != "" {
		entry = existing + ";" + entry
	}
	t.Setenv(key, entry)
}

func probePMVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "-v")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}
