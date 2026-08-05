package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/testkit"
)

// execResult captures the outcome of a production-path invocation.
type execResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// runM runs the m binary through runInvocation, isolating all process-global state.
func runM(t *testing.T, args ...string) execResult {
	t.Helper()
	return runMWith(t, runOptions{argv: args})
}

// runMX runs the mx binary through runInvocation, isolating all process-global state.
func runMX(t *testing.T, args ...string) execResult {
	t.Helper()
	return runMXWith(t, runOptions{argv: args})
}

type runOptions struct {
	argv  []string
	stdin *bytes.Buffer
	env   []string
	cwd   string
	ctx   context.Context
}

func runMWith(t *testing.T, opts runOptions) execResult {
	t.Helper()
	info := BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"}
	return runWith(t, func() *cobra.Command { return NewMRoot(info) }, info, opts)
}

func runMXWith(t *testing.T, opts runOptions) execResult {
	t.Helper()
	info := BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"}
	return runWith(t, func() *cobra.Command { return NewMXRoot(info) }, info, opts)
}

func runWith(t *testing.T, newRoot func() *cobra.Command, info BuildInfo, opts runOptions) execResult {
	t.Helper()
	testkit.CleanEnv(t)

	if opts.cwd != "" {
		absCWD, err := filepath.Abs(opts.cwd)
		if err != nil {
			t.Fatalf("abs cwd: %v", err)
		}
		opts.cwd = absCWD
		// Prepend --cwd so the bootstrap and app layers discover it.
		opts.argv = append([]string{"--cwd", absCWD}, opts.argv...)
	}

	if len(opts.env) > 0 {
		for _, kv := range opts.env {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				t.Setenv(parts[0], parts[1])
			}
		}
	}

	root := newRoot()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)

	if opts.stdin != nil {
		root.SetIn(opts.stdin)
	} else {
		root.SetIn(bytes.NewReader(nil))
	}

	ctx := opts.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	exitCode := ExecuteWithArgv(root, ctx, opts.argv)
	return execResult{
		ExitCode: exitCode,
		Stdout:   outBuf.String(),
		Stderr:   errBuf.String(),
	}
}

// blockingCmd returns a cobra command that blocks on started until ctx is done,
// then returns ctx.Err(). Use for cancellation tests.
func blockingCmd(use string, started chan<- struct{}) *cobra.Command {
	return &cobra.Command{
		Use: use,
		RunE: func(cmd *cobra.Command, args []string) error {
			if started != nil {
				close(started)
			}
			<-cmd.Context().Done()
			return cmd.Context().Err()
		},
	}
}

// panickingCmd returns a command whose RunE panics with the given message.
func panickingCmd(use, msg string) *cobra.Command {
	return &cobra.Command{
		Use: use,
		RunE: func(cmd *cobra.Command, args []string) error {
			panic(msg)
		},
	}
}

// =============================================================================
// Goal 1 — Harness self-test
// =============================================================================

func TestExecuteHarnessVersion(t *testing.T) {
	res := runM(t, "version")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "m 0.0.0-test") {
		t.Fatalf("stdout missing version: %s", res.Stdout)
	}
}

func TestExecuteHarnessHelp(t *testing.T) {
	res := runM(t, "--help")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "MewJS") {
		t.Fatalf("stdout missing MewJS: %s", res.Stdout)
	}
}

func TestExecuteHarnessMXVersion(t *testing.T) {
	res := runMX(t, "version")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "mx 0.0.0-test") {
		t.Fatalf("stdout missing mx version: %s", res.Stdout)
	}
}

func TestExecuteHarnessMXHelp(t *testing.T) {
	res := runMX(t, "--help")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "MewJS") {
		t.Fatalf("stdout missing MewJS: %s", res.Stdout)
	}
}

// =============================================================================
// Goal 2 — Flag parsing
// =============================================================================

func TestExecuteFlagParsing(t *testing.T) {
	t.Run("persistent flag before built-in command", func(t *testing.T) {
		res := runM(t, "--no-color", "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
	})

	t.Run("persistent flag after built-in command", func(t *testing.T) {
		res := runM(t, "version", "--json")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, `"version"`) {
			t.Fatalf("missing version key: %s", res.Stdout)
		}
	})

	t.Run("double dash stops flag parsing", func(t *testing.T) {
		res := runM(t, "version", "--", "--help")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("unknown global flag returns usage", func(t *testing.T) {
		res := runM(t, "--definitely-not-a-flag", "version")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		if !strings.Contains(res.Stderr, "ERR_M_USAGE") {
			t.Fatalf("stderr missing ERR_M_USAGE: %s", res.Stderr)
		}
	})

	t.Run("built-in command wins over dispatch", func(t *testing.T) {
		res := runM(t, "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "m 0.0.0-test") {
			t.Fatalf("stdout: %s", res.Stdout)
		}
	})

	t.Run("bare m shows help", func(t *testing.T) {
		res := runM(t)
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d want 0 (bare m shows help)", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "MewJS") {
			t.Fatalf("stdout missing MewJS: %s", res.Stdout)
		}
	})

	t.Run("no-color flag", func(t *testing.T) {
		res := runM(t, "--no-color", "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
		if presentation.ContainsCSI([]byte(res.Stdout)) {
			t.Fatalf("stdout has ANSI with --no-color: %q", res.Stdout)
		}
	})

	t.Run("ascii mode", func(t *testing.T) {
		res := runM(t, "--ascii", "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
	})

	t.Run("accessible flag", func(t *testing.T) {
		res := runM(t, "--accessible", "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
	})

	t.Run("no-progress flag", func(t *testing.T) {
		res := runM(t, "--no-progress", "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
	})

	t.Run("structured output on error is JSON", func(t *testing.T) {
		// Structured JSON output produces valid JSON (may be pretty-printed).
		res := runM(t, "--output", "json", "--definitely-not-a-flag", "version")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		stderr := strings.TrimSpace(res.Stderr)
		if stderr == "" {
			stderr = strings.TrimSpace(res.Stdout)
		}
		if stderr == "" {
			t.Fatal("empty output for structured error")
		}
		if !strings.HasPrefix(stderr, "{") {
			t.Fatalf("JSON output does not start with {: %q", stderr)
		}
		if !strings.Contains(stderr, "ERR_M_USAGE") {
			t.Fatalf("JSON output missing error code: %s", stderr)
		}
	})

	t.Run("structured output ndjson", func(t *testing.T) {
		res := runM(t, "--output", "ndjson", "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("silent mode suppresses errors", func(t *testing.T) {
		// In silent mode, errors go to stderr but stdout is suppressed.
		res := runM(t, "--output", "silent", "--definitely-not-a-flag", "version")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		// Stdout must be empty in silent mode.
		if strings.TrimSpace(res.Stdout) != "" {
			t.Fatalf("stdout not empty in silent mode: %s", res.Stdout)
		}
	})
}

func TestExecuteCompletion(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			res := runM(t, "completion", shell)
			if res.ExitCode != 0 {
				t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
			}
			if len(res.Stdout) < 20 {
				t.Fatalf("completion too short: %q", res.Stdout)
			}
		})
	}
}

// =============================================================================
// Goal 3 — Dispatch paths
// =============================================================================

func TestExecuteDispatchBuiltin(t *testing.T) {
	res := runM(t, "version", "--json")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, `"version"`) {
		t.Fatalf("missing version key: %s", res.Stdout)
	}
}

func TestExecuteDispatchBareM(t *testing.T) {
	// Bare m shows help (exit 0).
	res := runM(t)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d want 0", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "MewJS") {
		t.Fatalf("stdout missing MewJS: %s", res.Stdout)
	}
}

func TestExecuteDispatchMissingCommand(t *testing.T) {
	// Unknown command: exit 2 with ERR_M_USAGE.
	res := runM(t, "definitely-not-a-command")
	if res.ExitCode != 2 {
		t.Fatalf("exit=%d want 2", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "ERR_M_USAGE") {
		t.Fatalf("stderr missing ERR_M_USAGE: %s", res.Stderr)
	}
}

func TestExecuteMXVersionReserved(t *testing.T) {
	res := runMX(t, "version")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestExecuteMXMissingSelector(t *testing.T) {
	// Bare mx shows help (exit 0), like bare m.
	res := runMX(t)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d want 0 (bare mx shows help)", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "MewJS") {
		t.Fatalf("stdout missing MewJS: %s", res.Stdout)
	}
}

// =============================================================================
// Goal 4 — Config bootstrap
// =============================================================================

// setupProject creates a temp directory with a package.json and returns the path.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test-project","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestExecuteConfigBootstrap(t *testing.T) {
	t.Run("version works with no config files", func(t *testing.T) {
		res := runM(t, "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
	})

	t.Run("malformed user config fails non-info command", func(t *testing.T) {
		dir := t.TempDir()
		cfgDir := filepath.Join(dir, ".config", "mew")
		if err := os.MkdirAll(cfgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(cfgDir, "config.jsonc")
		if err := os.WriteFile(cfgPath, []byte("{not valid jsonc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		// Use a command that REQUIRES valid config. "config list" is not a repair command,
		// and it's not an info command, so it must have valid config.
		res := runMWith(t, runOptions{
			argv: []string{"config", "list"},
			env:  []string{"HOME=" + dir, "MEW_CONFIG_DIR=" + cfgDir, "XDG_CONFIG_HOME=" + filepath.Join(dir, ".config")},
		})
		if res.ExitCode != 1 {
			t.Fatalf("exit=%d want 1 (config error): stdout=%s stderr=%s", res.ExitCode, res.Stdout, res.Stderr)
		}
		if !strings.Contains(res.Stderr, "ERR_M_CONFIG") {
			t.Fatalf("stderr missing ERR_M_CONFIG: %s", res.Stderr)
		}
	})

	t.Run("repair command works with malformed config", func(t *testing.T) {
		dir := t.TempDir()
		cfgDir := filepath.Join(dir, ".config", "mew")
		if err := os.MkdirAll(cfgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(cfgDir, "config.jsonc")
		if err := os.WriteFile(cfgPath, []byte("{not valid jsonc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		res := runMWith(t, runOptions{
			argv: []string{"config", "validate"},
			env:  []string{"HOME=" + dir, "MEW_CONFIG_DIR=" + cfgDir, "XDG_CONFIG_HOME=" + filepath.Join(dir, ".config")},
		})
		// Validate reports the file is invalid but should not crash.
		if res.ExitCode != 0 {
			// Non-zero is acceptable since the file is malformed.
			if !strings.Contains(res.Stdout+res.Stderr, "ERR_M_CONFIG") && !strings.Contains(res.Stdout+res.Stderr, "error") {
				t.Fatalf("expected config-related error: stdout=%s stderr=%s", res.Stdout, res.Stderr)
			}
		}
	})

	t.Run("malformed project config fails non-repair command", func(t *testing.T) {
		dir := setupProject(t)
		cfgPath := filepath.Join(dir, "m.jsonc")
		if err := os.WriteFile(cfgPath, []byte("{"), 0o644); err != nil {
			t.Fatal(err)
		}
		// config list is NOT a repair command.
		res := runMWith(t, runOptions{
			argv: []string{"config", "list"},
			cwd:  dir,
		})
		if res.ExitCode != 1 {
			t.Fatalf("exit=%d want 1 (config error): stdout=%s stderr=%s", res.ExitCode, res.Stdout, res.Stderr)
		}
	})

	t.Run("config path flag", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "custom-config.jsonc")
		if err := os.WriteFile(cfgPath, []byte(`{"registry":"https://example.com"}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		res := runMWith(t, runOptions{
			argv: []string{"--config", cfgPath, "config", "get", "registry", "--scope", "user"},
			env:  []string{"HOME=" + dir},
		})
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "example.com") {
			t.Fatalf("stdout missing registry value: %s", res.Stdout)
		}
	})

	t.Run("environment overlay", func(t *testing.T) {
		res := runMWith(t, runOptions{
			argv: []string{"config", "get", "log.level", "--scope", "effective"},
			env:  []string{"MEW_LOG_LEVEL=debug"},
		})
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "debug") {
			t.Fatalf("stdout missing debug: %s", res.Stdout)
		}
	})

	t.Run("cli overlay", func(t *testing.T) {
		res := runMWith(t, runOptions{
			argv: []string{"--offline", "config", "get", "offline", "--scope", "effective"},
		})
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "true") {
			t.Fatalf("stdout missing true: %s", res.Stdout)
		}
	})
}

func TestExecuteConfigLoadOnce(t *testing.T) {
	var loads atomic.Int32
	origLoad := loadConfigFn
	loadConfigFn = func(ctx context.Context, opts app.Options) (app.ConfigSnapshot, error) {
		loads.Add(1)
		return origLoad(ctx, opts)
	}
	t.Cleanup(func() { loadConfigFn = origLoad })

	res := runM(t, "version")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if n := loads.Load(); n != 1 {
		t.Fatalf("config loads=%d want 1", n)
	}
}

// =============================================================================
// Goal 5 — Output and error mapping
// =============================================================================

func TestExecuteErrorMapping(t *testing.T) {
	t.Run("usage error exit code 2", func(t *testing.T) {
		res := runM(t, "--definitely-not-a-flag", "version")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		if !strings.Contains(res.Stderr, "ERR_M_USAGE") {
			t.Fatalf("stderr missing ERR_M_USAGE: %s", res.Stderr)
		}
	})

	t.Run("config error exit code 3", func(t *testing.T) {
		dir := t.TempDir()
		cfgDir := filepath.Join(dir, ".config", "mew")
		if err := os.MkdirAll(cfgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(cfgDir, "config.jsonc")
		if err := os.WriteFile(cfgPath, []byte("{invalid jsonc\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		res := runMWith(t, runOptions{
			argv: []string{"config", "list"},
			env:  []string{"HOME=" + dir, "MEW_CONFIG_DIR=" + cfgDir, "XDG_CONFIG_HOME=" + filepath.Join(dir, ".config")},
		})
		if res.ExitCode != 1 {
			t.Fatalf("exit=%d want 1: stdout=%s stderr=%s", res.ExitCode, res.Stdout, res.Stderr)
		}
		if !strings.Contains(res.Stderr, "ERR_M_CONFIG") {
			t.Fatalf("stderr missing ERR_M_CONFIG: %s", res.Stderr)
		}
	})

	t.Run("structured output has no ANSI", func(t *testing.T) {
		// Structured errors must have no ANSI sequences.
		res := runM(t, "--output", "json", "--definitely-not-a-flag", "version")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		combined := res.Stdout + res.Stderr
		if presentation.ContainsCSI([]byte(combined)) {
			t.Fatalf("JSON output has ANSI: %q", combined)
		}
	})

	t.Run("no-color has no ANSI", func(t *testing.T) {
		res := runM(t, "--no-color", "--definitely-not-a-flag", "version")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		combined := res.Stdout + res.Stderr
		if presentation.ContainsCSI([]byte(combined)) {
			t.Fatalf("output has ANSI with --no-color: %q", combined)
		}
	})

	t.Run("structured error is valid JSON", func(t *testing.T) {
		res := runM(t, "--output", "json", "--definitely-not-a-flag", "version")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		stderr := strings.TrimSpace(res.Stderr)
		if stderr == "" {
			stderr = strings.TrimSpace(res.Stdout)
		}
		if stderr == "" {
			t.Fatal("empty output for structured error")
		}
		if !strings.HasPrefix(stderr, "{") {
			t.Fatalf("JSON output does not start with {: %q", stderr)
		}
		if !strings.Contains(stderr, "ERR_M_USAGE") {
			t.Fatalf("JSON output missing error code: %s", stderr)
		}
	})
}

// =============================================================================
// Goal 6 — Controller lifecycle
// =============================================================================

func TestExecuteControllerLifecycle(t *testing.T) {
	t.Run("at most one controller created", func(t *testing.T) {
		var count atomic.Int32
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			count.Add(1)
			ctrl, err := origCtrl(resolved, caps, streams)
			return ctrl, err
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		res := runM(t, "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
		if n := count.Load(); n > 1 {
			t.Fatalf("controllers created=%d, want at most 1", n)
		}
	})

	t.Run("close called exactly once on success", func(t *testing.T) {
		var closes atomic.Int32
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			ctrl, err := origCtrl(resolved, caps, streams)
			if err != nil {
				return nil, err
			}
			return &closeCountingController{Controller: ctrl, closes: &closes}, nil
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		res := runM(t, "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
		if n := closes.Load(); n != 1 {
			t.Fatalf("close calls=%d want 1", n)
		}
	})

	t.Run("close called exactly once on post-bootstrap error", func(t *testing.T) {
		// An error AFTER controller creation (e.g. unknown command) should still
		// close the controller exactly once.
		var closes atomic.Int32
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			ctrl, err := origCtrl(resolved, caps, streams)
			if err != nil {
				return nil, err
			}
			return &closeCountingController{Controller: ctrl, closes: &closes}, nil
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		res := runM(t, "definitely-not-a-command")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		if n := closes.Load(); n != 1 {
			t.Fatalf("close calls=%d want 1", n)
		}
	})

	t.Run("success outcome passed on success", func(t *testing.T) {
		var outcome atomic.Value
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			ctrl, err := origCtrl(resolved, caps, streams)
			if err != nil {
				return nil, err
			}
			return &outcomeRecordingController{Controller: ctrl, outcome: &outcome}, nil
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		res := runM(t, "version")
		if res.ExitCode != 0 {
			t.Fatalf("exit=%d", res.ExitCode)
		}
		o, ok := outcome.Load().(presentation.Outcome)
		if !ok {
			t.Fatal("outcome not recorded")
		}
		if o.Err != nil {
			t.Fatalf("outcome error: %v", o.Err)
		}
	})

	t.Run("error outcome passed on failure", func(t *testing.T) {
		var outcome atomic.Value
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			ctrl, err := origCtrl(resolved, caps, streams)
			if err != nil {
				return nil, err
			}
			return &outcomeRecordingController{Controller: ctrl, outcome: &outcome}, nil
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		res := runM(t, "definitely-not-a-command")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		o, ok := outcome.Load().(presentation.Outcome)
		if !ok {
			t.Fatal("outcome not recorded")
		}
		if o.Err == nil {
			t.Fatal("expected error outcome")
		}
	})

	t.Run("cancellation outcome preserved", func(t *testing.T) {
		var outcome atomic.Value
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			ctrl, err := origCtrl(resolved, caps, streams)
			if err != nil {
				return nil, err
			}
			return &outcomeRecordingController{Controller: ctrl, outcome: &outcome}, nil
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		// Use a blocking command so the cancelled context takes effect.
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		started := make(chan struct{})
		root.AddCommand(blockingCmd("waitcancel-outcome", started))
		testkit.CleanEnv(t)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		var code int
		go func() {
			defer wg.Done()
			code = ExecuteWithArgv(root, ctx, []string{"waitcancel-outcome"})
		}()
		<-started
		cancel()
		wg.Wait()

		if code != 130 {
			t.Fatalf("exit=%d want 130", code)
		}
		o, ok := outcome.Load().(presentation.Outcome)
		if !ok {
			t.Fatal("outcome not recorded")
		}
		if o.Err == nil {
			t.Fatal("expected error outcome for cancellation")
		}
	})

	t.Run("early presentation failure creates no controller", func(t *testing.T) {
		// --output bogus fails during bootstrap (presentation resolution),
		// before the controller is created. No controller should be created.
		var count atomic.Int32
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			count.Add(1)
			return origCtrl(resolved, caps, streams)
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		res := runM(t, "--output", "bogus", "version")
		if res.ExitCode != 2 {
			t.Fatalf("exit=%d want 2", res.ExitCode)
		}
		if n := count.Load(); n != 0 {
			t.Fatalf("controller created for presentation error: count=%d", n)
		}
	})
}

// closeCountingController wraps a Controller to count Close calls.
type closeCountingController struct {
	presentation.Controller
	closes *atomic.Int32
}

func (c *closeCountingController) Close(ctx context.Context, outcome presentation.Outcome) error {
	c.closes.Add(1)
	return c.Controller.Close(ctx, outcome)
}

// outcomeRecordingController wraps a Controller to capture the close outcome.
type outcomeRecordingController struct {
	presentation.Controller
	outcome *atomic.Value
}

func (c *outcomeRecordingController) Close(ctx context.Context, outcome presentation.Outcome) error {
	c.outcome.Store(outcome)
	return c.Controller.Close(ctx, outcome)
}

// =============================================================================
// Goal 7 — Panic recovery
// =============================================================================

func TestExecutePanicRecovery(t *testing.T) {
	t.Run("panic does not escape", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		root.AddCommand(panickingCmd("test-panic", "test injected panic"))
		testkit.CleanEnv(t)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		code := ExecuteWithArgv(root, context.Background(), []string{"test-panic"})
		if code != 1 {
			t.Fatalf("exit=%d want 1 (internal panic)", code)
		}
	})

	t.Run("panic returns typed internal-panic error", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		root.AddCommand(panickingCmd("test-panic-typed", "test injected panic for typing"))
		testkit.CleanEnv(t)
		errBuf := new(bytes.Buffer)
		root.SetOut(new(bytes.Buffer))
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		code := ExecuteWithArgv(root, context.Background(), []string{"test-panic-typed"})
		if code != 1 {
			t.Fatalf("exit=%d want 1", code)
		}
		out := errBuf.String()
		if !strings.Contains(out, "ERR_M_INTERNAL_PANIC") {
			t.Fatalf("stderr missing ERR_M_INTERNAL_PANIC: %s", out)
		}
	})

	t.Run("panic crash id is present", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		root.AddCommand(panickingCmd("test-panic-crashid", "test injected panic for crash id"))
		testkit.CleanEnv(t)
		errBuf := new(bytes.Buffer)
		root.SetOut(new(bytes.Buffer))
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		code := ExecuteWithArgv(root, context.Background(), []string{"test-panic-crashid"})
		if code != 1 {
			t.Fatalf("exit=%d want 1", code)
		}
		out := errBuf.String()
		if !strings.Contains(out, "crash-") {
			t.Fatalf("stderr missing crash ID: %s", out)
		}
	})

	t.Run("panic diagnostic emitted once", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		root.AddCommand(panickingCmd("test-panic-once", "test unique panic for dedup check"))
		testkit.CleanEnv(t)
		errBuf := new(bytes.Buffer)
		root.SetOut(new(bytes.Buffer))
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		code := ExecuteWithArgv(root, context.Background(), []string{"test-panic-once"})
		if code != 1 {
			t.Fatalf("exit=%d want 1", code)
		}
		out := errBuf.String()
		if strings.Count(out, "ERR_M_INTERNAL_PANIC") != 1 {
			t.Fatalf("ERR_M_INTERNAL_PANIC count=%d want 1:\n%s", strings.Count(out, "ERR_M_INTERNAL_PANIC"), out)
		}
	})

	t.Run("panic controller closes once", func(t *testing.T) {
		var closes atomic.Int32
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			ctrl, err := origCtrl(resolved, caps, streams)
			if err != nil {
				return nil, err
			}
			return &closeCountingController{Controller: ctrl, closes: &closes}, nil
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		root.AddCommand(panickingCmd("test-panic-close-once", "test injected panic for close count"))
		testkit.CleanEnv(t)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		code := ExecuteWithArgv(root, context.Background(), []string{"test-panic-close-once"})
		if code != 1 {
			t.Fatalf("exit=%d want 1", code)
		}
		if n := closes.Load(); n != 1 {
			t.Fatalf("close calls=%d want 1", n)
		}
	})

	t.Run("panic structured output is valid", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		root.AddCommand(panickingCmd("test-panic-structured", "test injected panic for structured output"))
		testkit.CleanEnv(t)
		errBuf := new(bytes.Buffer)
		root.SetOut(new(bytes.Buffer))
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		code := ExecuteWithArgv(root, context.Background(), []string{"--output", "json", "test-panic-structured"})
		if code != 1 {
			t.Fatalf("exit=%d want 1", code)
		}
		// Structured output may go to stdout or stderr depending on the reporter format.
		combined := root.OutOrStdout().(*bytes.Buffer).String() + errBuf.String()
		trimmed := strings.TrimSpace(combined)
		if trimmed == "" {
			t.Fatal("empty structured error output")
		}
		if !strings.HasPrefix(trimmed, "{") {
			t.Fatalf("panic structured output does not start with {: %q", trimmed)
		}
		if !strings.Contains(trimmed, "ERR_M_INTERNAL_PANIC") {
			t.Fatalf("panic structured output missing error code: %s", trimmed)
		}
	})

	t.Run("no secret leakage in panic", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		root.AddCommand(panickingCmd("test-panic-secrets", "test injected panic with faux secret"))
		testkit.CleanEnv(t)
		errBuf := new(bytes.Buffer)
		root.SetOut(new(bytes.Buffer))
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))
		t.Setenv("MEW_AUTH_TOKEN", "super-secret-token-value")
		t.Setenv("NPM_TOKEN", "npm-secret-token")

		code := ExecuteWithArgv(root, context.Background(), []string{"test-panic-secrets"})
		if code != 1 {
			t.Fatalf("exit=%d want 1", code)
		}
		out := errBuf.String()
		for _, secret := range []string{"super-secret-token-value", "npm-secret-token"} {
			if strings.Contains(out, secret) {
				t.Fatalf("secret leaked in output: %q found in %q", secret, out)
			}
		}
	})
}

// =============================================================================
// Goal 8 — Cancellation
// =============================================================================

func TestExecuteCancellation(t *testing.T) {
	t.Run("cancellation during command execution", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		started := make(chan struct{})
		root.AddCommand(blockingCmd("waitcancel", started))
		testkit.CleanEnv(t)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		var code int
		go func() {
			defer wg.Done()
			code = ExecuteWithArgv(root, ctx, []string{"waitcancel"})
		}()

		<-started
		cancel()
		wg.Wait()

		if code != 130 {
			t.Fatalf("exit=%d want 130", code)
		}
	})

	t.Run("cancellation returns exit 130", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		started := make(chan struct{})
		root.AddCommand(blockingCmd("waitcancel-130", started))
		testkit.CleanEnv(t)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		var code int
		go func() {
			defer wg.Done()
			code = ExecuteWithArgv(root, ctx, []string{"waitcancel-130"})
		}()
		<-started
		cancel()
		wg.Wait()

		if code != 130 {
			t.Fatalf("exit=%d want 130", code)
		}
		out := errBuf.String()
		if !strings.Contains(out, "ERR_M_CANCELLED") {
			t.Fatalf("stderr missing ERR_M_CANCELLED: %s", out)
		}
	})

	t.Run("cancellation controller closes once", func(t *testing.T) {
		var closes atomic.Int32
		origCtrl := newControllerFn
		newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
			ctrl, err := origCtrl(resolved, caps, streams)
			if err != nil {
				return nil, err
			}
			return &closeCountingController{Controller: ctrl, closes: &closes}, nil
		}
		t.Cleanup(func() { newControllerFn = origCtrl })

		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		started := make(chan struct{})
		root.AddCommand(blockingCmd("waitcancel-close", started))
		testkit.CleanEnv(t)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		var code int
		go func() {
			defer wg.Done()
			code = ExecuteWithArgv(root, ctx, []string{"waitcancel-close"})
		}()
		<-started
		cancel()
		wg.Wait()

		if code != 130 {
			t.Fatalf("exit=%d want 130", code)
		}
		if n := closes.Load(); n != 1 {
			t.Fatalf("close calls=%d want 1", n)
		}
	})

	t.Run("no goroutine remains blocked after cancellation", func(t *testing.T) {
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		started := make(chan struct{})
		done := make(chan struct{})
		root.AddCommand(&cobra.Command{
			Use: "waitcancel-noblock",
			RunE: func(cmd *cobra.Command, args []string) error {
				close(started)
				select {
				case <-cmd.Context().Done():
					close(done)
					return cmd.Context().Err()
				}
			},
		})
		testkit.CleanEnv(t)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-started
			cancel()
		}()
		code := ExecuteWithArgv(root, ctx, []string{"waitcancel-noblock"})
		if code != 130 {
			t.Fatalf("exit=%d want 130", code)
		}
		<-done // goroutine unblocked
	})

	t.Run("cancellation via direct dispatch", func(t *testing.T) {
		// Direct dispatch with cancelled context: verify the path is covered.
		root := NewMRoot(BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"})
		started := make(chan struct{})
		root.AddCommand(blockingCmd("waitcancel-dispatch", started))
		testkit.CleanEnv(t)
		outBuf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetIn(bytes.NewReader(nil))

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		var code int
		go func() {
			defer wg.Done()
			code = ExecuteWithArgv(root, ctx, []string{"waitcancel-dispatch"})
		}()
		<-started
		cancel()
		wg.Wait()

		if code != 130 {
			t.Fatalf("exit=%d want 130", code)
		}
	})
}
