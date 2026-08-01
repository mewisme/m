package runner_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/process"
	"github.com/mewisme/mew/internal/runner"
)

type recordSupervisor struct {
	specs    []process.Spec
	exitCode int
	waitErr  error
}

func (r *recordSupervisor) Start(_ context.Context, spec process.Spec) (*process.Handle, error) {
	r.specs = append(r.specs, spec)
	return &process.Handle{}, nil
}

func (r *recordSupervisor) Wait(context.Context, *process.Handle) error {
	if r.waitErr != nil {
		return r.waitErr
	}
	if r.exitCode != 0 {
		return &process.ExitError{Code: r.exitCode}
	}
	return nil
}

var _ process.ProcessSupervisor = (*recordSupervisor)(nil)

type captureReporter struct {
	events []diagnostics.Event
}

func (c *captureReporter) Progress(ev diagnostics.Event) { c.events = append(c.events, ev) }
func (c *captureReporter) Error(error)                   {}
func (c *captureReporter) Debug(string, ...diagnostics.Attr) {
}
func (c *captureReporter) WorkspaceTask(diagnostics.WorkspaceTaskEvent) {}
func (c *captureReporter) ChildOutput(diagnostics.ChildOutputEvent, diagnostics.WorkspaceOutputMode) {
}
func (c *captureReporter) WorkspaceSummary(diagnostics.WorkspaceSummaryEvent) {}
func (c *captureReporter) EnvironmentPrepared(diagnostics.EnvironmentPreparedEvent) error {
	return nil
}
func (c *captureReporter) OperationStarted(diagnostics.OperationStartedEvent)     {}
func (c *captureReporter) OperationProgress(diagnostics.OperationProgressEvent)   {}
func (c *captureReporter) OperationCompleted(diagnostics.OperationCompletedEvent) {}
func (c *captureReporter) Notice(diagnostics.NoticeEvent)                         {}

func baseRunOptions(scripts map[string]string, selector string) runner.RunOptions {
	return runner.RunOptions{
		ProjectRoot: "/proj",
		PackageDir:  "/proj",
		PackageName: "demo",
		PackageVer:  "1.0.0",
		Scripts:     scripts,
		Selector:    selector,
		HostEnv:     []string{"PATH=/bin"},
	}
}

func TestDefaultRunnerRunsHooksInOrder(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	scripts := map[string]string{
		"predev":  "echo pre",
		"dev":     "vite",
		"postdev": "echo post",
	}
	_, err := r.Run(context.Background(), baseRunOptions(scripts, "dev"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.specs) != 3 {
		t.Fatalf("got %d stages, want 3", len(rec.specs))
	}
	want := []string{"echo pre", "vite", "echo post"}
	for i, cmd := range want {
		if rec.specs[i].Path != cmd {
			t.Fatalf("stage %d command %q, want %q", i, rec.specs[i].Path, cmd)
		}
	}
}

func TestDefaultRunnerForwardsArgsOnlyToPrimaryScript(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	scripts := map[string]string{
		"prebuild":  "echo pre",
		"build":     "webpack",
		"postbuild": "echo post",
	}
	opts := baseRunOptions(scripts, "build")
	opts.ForwardedArgs = []string{"--mode", "production"}
	_, err := r.Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.specs) != 3 {
		t.Fatalf("got %d stages, want 3", len(rec.specs))
	}
	if rec.specs[0].Path != "echo pre" {
		t.Fatalf("pre command %q", rec.specs[0].Path)
	}
	wantPrimary := "webpack --mode production"
	if runtime.GOOS != "windows" {
		wantPrimary = "webpack '--mode' production"
	}
	if rec.specs[1].Path != wantPrimary {
		t.Fatalf("primary command %q, want %q", rec.specs[1].Path, wantPrimary)
	}
	if rec.specs[2].Path != "echo post" {
		t.Fatalf("post command %q", rec.specs[2].Path)
	}
}

func TestDefaultRunnerIfPresentMissingScript(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	opts := baseRunOptions(map[string]string{"dev": "vite"}, "start")
	opts.IfPresent = true
	res, err := r.Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.specs) != 0 {
		t.Fatalf("expected no execution, got %d specs", len(rec.specs))
	}
	if len(res.Plans) != 0 {
		t.Fatalf("expected empty plans, got %+v", res.Plans)
	}
}

func TestDefaultRunnerIfPresentDoesNotSwallowUsage(t *testing.T) {
	r := &runner.DefaultRunner{Supervisor: &recordSupervisor{}}
	opts := baseRunOptions(map[string]string{"dev": "vite"}, "/[unclosed")
	opts.IfPresent = true
	_, err := r.Run(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code %q, want usage", apperr.CodeOf(err))
	}
}

func TestDefaultRunnerChildExitSetsExitStatus(t *testing.T) {
	rec := &recordSupervisor{exitCode: 42}
	r := &runner.DefaultRunner{Supervisor: rec}
	_, err := r.Run(context.Background(), baseRunOptions(map[string]string{"dev": "vite"}, "dev"))
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.ExitCode(err) != 42 {
		t.Fatalf("exit=%d, want 42", apperr.ExitCode(err))
	}
	var es *apperr.ExitStatus
	if !errors.As(err, &es) || es.ExitCode() != 42 {
		t.Fatalf("expected ExitStatus with code 42, got %+v", err)
	}
	if apperr.CodeOf(err) != apperr.ChildExit {
		t.Fatalf("expected ChildExit code, got %s", apperr.CodeOf(err))
	}
}

func TestDefaultRunnerEmitsProgressEvents(t *testing.T) {
	rec := &recordSupervisor{}
	rep := &captureReporter{}
	r := &runner.DefaultRunner{Supervisor: rec}
	opts := baseRunOptions(map[string]string{
		"predev": "echo pre",
		"dev":    "vite",
	}, "dev")
	opts.Reporter = rep
	_, err := r.Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.events) != 2 {
		t.Fatalf("got %d events, want 2", len(rep.events))
	}
	for i, wantScript := range []string{"predev", "dev"} {
		ev := rep.events[i]
		if ev.Phase != "run" || ev.Type != "progress" || ev.V != 1 {
			t.Fatalf("event %d: %+v", i, ev)
		}
		if ev.Package != wantScript {
			t.Fatalf("event %d package %q, want %s", i, ev.Package, wantScript)
		}
	}
}

func TestDefaultRunnerWiresStdio(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	var stdin, stdout, stderr bytes.Buffer
	stdin.WriteString("in")
	opts := baseRunOptions(map[string]string{"dev": "vite"}, "dev")
	opts.Stdin = &stdin
	opts.Stdout = &stdout
	opts.Stderr = &stderr
	_, err := r.Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.specs) != 1 {
		t.Fatal("expected one stage")
	}
	if rec.specs[0].Stdin != io.Reader(&stdin) {
		t.Fatal("stdin not wired")
	}
	if rec.specs[0].Stdout != io.Writer(&stdout) {
		t.Fatal("stdout not wired")
	}
	if rec.specs[0].Stderr != io.Writer(&stderr) {
		t.Fatal("stderr not wired")
	}
}

func TestDefaultRunnerSetsLifecycleEnv(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	opts := baseRunOptions(map[string]string{"dev": "vite"}, "dev")
	opts.PackageJSON = "/proj/package.json"
	_, err := r.Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	env := strings.Join(rec.specs[0].Env, "\n")
	for _, want := range []string{
		"INIT_CWD=/proj",
		"npm_lifecycle_event=dev",
		"npm_lifecycle_script=vite",
		"npm_package_name=demo",
		"npm_package_version=1.0.0",
		"npm_package_json=/proj/package.json",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("env missing %q in:\n%s", want, env)
		}
	}
}

func TestDefaultRunnerRegexRunsMatchedScripts(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	scripts := map[string]string{
		"test:a": "a",
		"test:b": "b",
	}
	_, err := r.Run(context.Background(), baseRunOptions(scripts, `/^test:/`))
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.specs) != 2 {
		t.Fatalf("got %d stages, want 2", len(rec.specs))
	}
	if rec.specs[0].Path != "a" || rec.specs[1].Path != "b" {
		t.Fatalf("unexpected commands: %#v", []string{rec.specs[0].Path, rec.specs[1].Path})
	}
}

func TestNewDefaultRunnerUsesExecSupervisor(t *testing.T) {
	r := runner.NewDefaultRunner()
	if r == nil || r.Supervisor == nil {
		t.Fatal("expected non-nil runner with supervisor")
	}
}
