package conformance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunTest executes go test for one suite.
func RunTest(ctx context.Context, repoRoot string, suite Suite) (exitCode int, output string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	args := []string{"test", suite.Package, "-count=1"}
	if suite.Run != "." {
		args = append(args, "-run", suite.Run)
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	runErr := cmd.Run()
	output = strings.TrimSpace(buf.String())
	if runErr == nil {
		return 0, output, nil
	}
	var ee *exec.ExitError
	if errors.As(runErr, &ee) {
		return ee.ExitCode(), output, nil
	}
	return 1, output, runErr
}

// RepoRootFromModule walks up from start until go.mod is found.
func RepoRootFromModule(start string) (string, error) {
	dir := start
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	dir = abs
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("module root not found from %s", abs)
		}
		dir = parent
	}
}

func summarizeOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "go test failed"
	}
	lines := strings.Split(output, "\n")
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	return strings.Join(lines, "\n")
}

func suiteResultFromRun(suite Suite, started time.Time, exitCode int, output string, runErr error) SuiteResult {
	res := SuiteResult{
		ID:       suite.ID,
		Title:    suite.Title,
		Package:  suite.Package,
		Run:      suite.Run,
		Required: suite.Required,
		Duration: time.Since(started).Round(time.Millisecond).String(),
		ExitCode: exitCode,
	}
	if runErr != nil {
		res.Status = StatusFailed
		res.Error = runErr.Error()
		return res
	}
	if exitCode == 0 {
		res.Status = StatusPassed
		return res
	}
	res.Status = StatusFailed
	res.Error = summarizeOutput(output)
	return res
}
