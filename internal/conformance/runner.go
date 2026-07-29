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

// RunTest executes go test -json for one suite.
func RunTest(ctx context.Context, repoRoot string, suite Suite) (exitCode int, summary TestSummary, output string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	args := []string{"test", suite.Package, "-count=1", "-json"}
	if suite.Run != "." {
		args = append(args, "-run", suite.Run)
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if conformanceRequireTools() {
		cmd.Env = append(cmd.Env, "MEW_CONFORMANCE_REQUIRE_TOOLS=1")
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	combined := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		if combined != "" {
			combined += "\n"
		}
		combined += strings.TrimSpace(stderr.String())
	}
	output = combined
	summary, parseErr := ParseTestJSONLines(&stdout)
	if parseErr != nil {
		return exitCodeFromRun(runErr), summary, output, parseErr
	}
	return exitCodeFromRun(runErr), summary, output, nil
}

func exitCodeFromRun(runErr error) int {
	if runErr == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(runErr, &ee) {
		return ee.ExitCode()
	}
	return 1
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

func suiteResultFromRun(suite Suite, started time.Time, exitCode int, summary TestSummary, output string, runErr error) SuiteResult {
	res := SuiteResult{
		ID:           suite.ID,
		Title:        suite.Title,
		Package:      suite.Package,
		Run:          suite.Run,
		Required:     suite.Required,
		Duration:     time.Since(started).Round(time.Millisecond).String(),
		ExitCode:     exitCode,
		TestsMatched: summary.TestsMatched,
		Passed:       summary.Passed,
		Failed:       summary.Failed,
		Skipped:      summary.Skipped,
	}
	if runErr != nil {
		res.Status = StatusFailed
		if summary.ParseError != "" {
			res.Error = summary.ParseError
		} else {
			res.Error = runErr.Error()
		}
		return res
	}
	if reason := summary.FailReason(suite, exitCode, conformanceRequireTools()); reason != "" {
		res.Status = StatusFailed
		res.Error = reason
		return res
	}
	res.Status = StatusPassed
	return res
}
