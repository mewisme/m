package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// listTestsForSuite runs go test -list with the suite regex.
func listTestsForSuite(repoRoot string, suite RunnerSuite, extraEnv []string) ([]string, error) {
	timeout, err := time.ParseDuration(suite.Timeout)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{"test", "-list", suite.Run, suite.Package}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = buildSuiteEnv(repoRoot, suite, suiteIsolation{}, extraEnv)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("test-list preflight: %s", msg)
	}
	re, err := regexp.Compile(suite.Run)
	if err != nil {
		return nil, err
	}
	var matched []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "ok ") || strings.HasPrefix(line, "?") {
			continue
		}
		if re.MatchString(line) {
			matched = append(matched, line)
		}
	}
	sort.Strings(matched)
	return matched, nil
}

type suiteIsolation struct {
	Root        string
	CacheRoot   string
	ConfigRoot  string
	ArtifactDir string
	RegistryDir string
	Cleanup     func() error
}

func prepareSuiteIsolation(repoRoot string, suite RunnerSuite) (suiteIsolation, error) {
	if suite.Isolation != "fresh" {
		return suiteIsolation{}, apperr.New(apperr.Unsupported, "conformance.runner.isolation", suite.ID, "unsupported isolation")
	}
	root, err := os.MkdirTemp("", "mew-runner-suite-*")
	if err != nil {
		return suiteIsolation{}, apperr.Wrap(apperr.IO, "conformance.runner.isolation", suite.ID, err)
	}
	iso := suiteIsolation{
		Root:        root,
		CacheRoot:   filepath.Join(root, "cache"),
		ConfigRoot:  filepath.Join(root, "config"),
		ArtifactDir: filepath.Join(root, "artifacts"),
		RegistryDir: filepath.Join(repoRoot, "fixtures", "registry"),
		Cleanup: func() error {
			return os.RemoveAll(root)
		},
	}
	for _, dir := range []string{iso.CacheRoot, iso.ConfigRoot, iso.ArtifactDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			_ = iso.Cleanup()
			return suiteIsolation{}, apperr.Wrap(apperr.IO, "conformance.runner.isolation", suite.ID, err)
		}
	}
	return iso, nil
}

func buildSuiteEnv(repoRoot string, suite RunnerSuite, iso suiteIsolation, extra []string) []string {
	base := sanitizedInheritedEnv()
	base = append(base, extra...)
	base = append(base,
		"CGO_ENABLED=0",
		"MEW_CACHE_ROOT="+iso.CacheRoot,
		"MEW_CONFIG_ROOT="+iso.ConfigRoot,
		"MEW_TEST_ARTIFACT_DIR="+iso.ArtifactDir,
		"MEW_TEST_NETWORK_POLICY="+suite.NetworkPolicy,
	)
	if suite.NetworkPolicy == "local-fixture" {
		base = append(base, "MEW_TEST_REGISTRY_URL=file://"+filepath.ToSlash(iso.RegistryDir))
	}
	for k, v := range suite.Environment {
		base = append(base, k+"="+v)
	}
	_ = repoRoot
	return base
}

func sanitizedInheritedEnv() []string {
	stripPrefixes := []string{
		"NPM_", "PNPM_", "YARN_", "BUN_", "GH_", "GITLAB_", "AWS_", "AZURE_",
		"GOOGLE_", "DOCKER_", "SSH_", "MEW_REGISTRY_", "NODE_AUTH_",
	}
	var out []string
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		key := kv[:i]
		if _, allow := harnessEnvAllowlist[key]; allow {
			out = append(out, kv)
			continue
		}
		strip := false
		upper := strings.ToUpper(key)
		for _, p := range stripPrefixes {
			if strings.HasPrefix(upper, p) || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "PASSWORD") {
				strip = true
				break
			}
		}
		if !strip {
			out = append(out, kv)
		}
	}
	return out
}

type suiteRunOutput struct {
	ExitCode        int
	Summary         TestSummary
	Stdout          string
	Stderr          string
	StdoutTruncated bool
	StderrTruncated bool
	MatchedTests    []string
	SkippedTests    []string
	RunErr          error
}

func runRunnerSuite(ctx context.Context, repoRoot string, suite RunnerSuite, iso suiteIsolation) suiteRunOutput {
	timeout, _ := time.ParseDuration(suite.Timeout)
	external := timeout + RunnerExternalDeadlineGrace
	runCtx, cancel := context.WithTimeout(ctx, external)
	defer cancel()

	args := []string{
		"test", "-json", "-count=1", "-shuffle=off",
		"-timeout=" + suite.Timeout,
		"-run=" + suite.Run,
		suite.Package,
	}
	cmd := exec.CommandContext(runCtx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = buildSuiteEnv(repoRoot, suite, iso, nil)

	stdoutCap := newLimitedBuffer(RunnerMaxStdoutBytes, RunnerDiagnosticTail)
	stderrCap := newLimitedBuffer(RunnerMaxStderrBytes, RunnerDiagnosticTail)
	cmd.Stdout = stdoutCap
	cmd.Stderr = stderrCap

	runErr := cmd.Run()
	exitCode := exitCodeFromRun(runErr)
	summary, parseErr := ParseRunnerTestJSON(stdoutCap.Bytes(), suite)
	if parseErr != nil {
		if summary.ParseError == "" {
			summary.ParseError = parseErr.Error()
		}
	}
	matched, _ := listTestsForSuite(repoRoot, suite, buildSuiteEnv(repoRoot, suite, iso, nil))
	return suiteRunOutput{
		ExitCode:        exitCode,
		Summary:         summary,
		Stdout:          stdoutCap.String(),
		Stderr:          stderrCap.String(),
		StdoutTruncated: stdoutCap.Truncated(),
		StderrTruncated: stderrCap.Truncated(),
		MatchedTests:    matched,
		SkippedTests:    summary.SkippedTests,
		RunErr:          runErr,
	}
}

type limitedBuffer struct {
	maxBytes  int
	tailKeep  int
	buf       bytes.Buffer
	truncated bool
}

func newLimitedBuffer(maxBytes, tailKeep int) *limitedBuffer {
	return &limitedBuffer{maxBytes: maxBytes, tailKeep: tailKeep}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) <= b.maxBytes {
		return b.buf.Write(p)
	}
	b.truncated = true
	remain := b.maxBytes - b.buf.Len()
	if remain > 0 {
		_, _ = b.buf.Write(p[:remain])
	}
	// ponytail: keep prefix+tail only; full stream not needed for cert diagnostics
	all := b.buf.Bytes()
	if len(all) > b.tailKeep {
		tail := append([]byte(nil), all[len(all)-b.tailKeep:]...)
		b.buf.Reset()
		_, _ = b.buf.Write(all[:min(len(all), b.maxBytes-b.tailKeep)])
		_, _ = b.buf.Write(tail)
	}
	return len(p), nil
}

func (b *limitedBuffer) Bytes() []byte   { return b.buf.Bytes() }
func (b *limitedBuffer) String() string  { return b.buf.String() }
func (b *limitedBuffer) Truncated() bool { return b.truncated }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ParseRunnerTestJSON parses go test -json with runner skip enforcement.
func ParseRunnerTestJSON(data []byte, suite RunnerSuite) (TestSummary, error) {
	summary, err := ParseTestJSON(data)
	if err != nil {
		return summary, err
	}
	summary.SkippedTests = collectSkippedTopLevelTests(data)
	if len(summary.SkippedTests) > 0 {
		if suite.Probe {
			return summary, nil
		}
		if suite.Required {
			summary.ParseError = fmt.Sprintf("required suite has skipped tests: %s", strings.Join(summary.SkippedTests, ", "))
			return summary, fmt.Errorf("%s", summary.ParseError)
		}
	}
	return summary, nil
}

func collectSkippedTopLevelTests(data []byte) []string {
	type ev struct {
		Action string `json:"Action"`
		Test   string `json:"Test"`
	}
	seen := map[string]struct{}{}
	var skipped []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || len(line) > RunnerMaxJSONEventSize {
			continue
		}
		var e ev
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		if e.Action != "skip" || e.Test == "" {
			continue
		}
		top := e.Test
		if i := strings.Index(top, "/"); i >= 0 {
			top = top[:i]
		}
		if _, ok := seen[top]; ok {
			continue
		}
		seen[top] = struct{}{}
		skipped = append(skipped, top)
	}
	sort.Strings(skipped)
	return skipped
}
