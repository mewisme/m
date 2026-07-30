package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/process"
)

// DefaultRunner executes package scripts via process.ExecSupervisor.
type DefaultRunner struct {
	Supervisor process.ProcessSupervisor
}

// NewDefaultRunner returns a runner backed by process.NewExecSupervisor.
func NewDefaultRunner() *DefaultRunner {
	return &DefaultRunner{Supervisor: process.NewExecSupervisor()}
}

var _ ScriptRunner = (*DefaultRunner)(nil)

// Run resolves the selector, expands hooks, and runs each stage sequentially.
func (r *DefaultRunner) Run(ctx context.Context, opts RunOptions) (RunResult, error) {
	names, err := Lookup(opts.Scripts, opts.Selector)
	if err != nil {
		if opts.IfPresent && apperr.CodeOf(err) == apperr.NotFound {
			return RunResult{}, nil
		}
		return RunResult{}, err
	}

	plans := ExpandPlans(opts.Scripts, names)
	if len(plans) == 0 {
		if opts.IfPresent {
			return RunResult{}, nil
		}
		return RunResult{}, apperr.New(apperr.NotFound, "runner.run", opts.Selector, "missing script")
	}

	sup := r.supervisor()
	hostEnv := opts.HostEnv
	if len(hostEnv) == 0 {
		hostEnv = os.Environ()
	}
	pkgJSON := opts.PackageJSON
	if pkgJSON == "" {
		pkgJSON = PackageJSONPath(opts.PackageDir)
	}
	stdin, stdout, stderr := opts.Stdin, opts.Stdout, opts.Stderr
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	result := RunResult{Plans: plans}
	for _, plan := range plans {
		for _, stage := range plan.Stages {
			emitRunProgress(opts.Reporter, stage.Event)

			script := stage.Script
			if stage.Event == plan.Name && len(opts.ForwardedArgs) > 0 {
				script = appendForwardedArgs(script, opts.ForwardedArgs)
			}

			env := BuildEnv(EnvOptions{
				HostEnv:     hostEnv,
				InitCWD:     opts.ProjectRoot,
				PackageDir:  opts.PackageDir,
				NodeModules: opts.NodeModules,
				PackageJSON: pkgJSON,
				PackageName: opts.PackageName,
				PackageVer:  opts.PackageVer,
				Event:       stage.Event,
				Script:      stage.Script,
			})

			code, runErr := runStage(ctx, sup, runStageInput{
				command: script,
				dir:     env.Dir,
				env:     env.Vars,
				stdin:   stdin,
				stdout:  stdout,
				stderr:  stderr,
				subject: stage.Event,
				suspend: opts.Suspend,
				resume:  opts.Resume,
			})
			result.ExitCode = code
			if runErr != nil {
				return result, runErr
			}
		}
	}
	return result, nil
}

type runStageInput struct {
	command string
	dir     string
	env     []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	subject string
	suspend func(context.Context) error
	resume  func(context.Context) error
}

func (r *DefaultRunner) supervisor() process.ProcessSupervisor {
	if r != nil && r.Supervisor != nil {
		return r.Supervisor
	}
	return process.NewExecSupervisor()
}

func runStage(ctx context.Context, sup process.ProcessSupervisor, in runStageInput) (int, error) {
	shell := ""
	if v, ok := lookupEnv(in.env, "ComSpec"); ok {
		shell = v
	}
	spec := process.Spec{
		Path:   in.command,
		Dir:    in.dir,
		Env:    in.env,
		Shell:  shell,
		Stdin:  in.stdin,
		Stdout: in.stdout,
		Stderr: in.stderr,
	}
	if in.suspend != nil {
		_ = in.suspend(ctx)
	}
	if in.resume != nil {
		defer func() { _ = in.resume(ctx) }()
	}
	h, err := sup.Start(ctx, spec)
	if err != nil {
		return 1, apperr.Wrap(apperr.IO, "runner.run", in.subject, err)
	}
	waitErr := sup.Wait(ctx, h)
	if errors.Is(ctx.Err(), context.Canceled) {
		return process.ExitCode(waitErr), apperr.Wrap(apperr.Cancelled, "runner.run", in.subject, context.Canceled)
	}
	code := process.ExitCode(waitErr)
	if code != 0 {
		ae := apperr.New(apperr.Internal, "runner.run", in.subject,
			fmt.Sprintf("script %s exited with code %d", in.subject, code))
		ae.ExitHint = code
		return code, ae
	}
	if waitErr != nil {
		return 1, apperr.Wrap(apperr.IO, "runner.run", in.subject, waitErr)
	}
	return 0, nil
}

func emitRunProgress(rep diagnostics.Reporter, script string) {
	if rep == nil {
		return
	}
	rep.Progress(diagnostics.Event{
		V:       1,
		Type:    "progress",
		Phase:   "run",
		Package: script,
	})
}

func appendForwardedArgs(cmd string, args []string) string {
	if len(args) == 0 {
		return cmd
	}
	var b strings.Builder
	b.WriteString(cmd)
	for _, arg := range args {
		b.WriteByte(' ')
		b.WriteString(quoteShellArg(arg))
	}
	return b.String()
}

func needsUnixShellQuote(arg string) bool {
	if arg == "" {
		return true
	}
	if strings.HasPrefix(arg, "-") {
		return true
	}
	const special = " \t\"'&|;<>()$`\\*?[]#~!"
	return strings.ContainsAny(arg, special)
}
func quoteShellArg(arg string) string {
	if runtime.GOOS == "windows" {
		if arg == "" {
			return `""`
		}
		if !strings.ContainsAny(arg, " \t\"&|<>^") {
			return arg
		}
		return `"` + strings.ReplaceAll(arg, `"`, `""`) + `"`
	}
	if !needsUnixShellQuote(arg) {
		return arg
	}
	if !strings.Contains(arg, "'") {
		return "'" + arg + "'"
	}
	return "'" + strings.ReplaceAll(arg, "'", `'"'"'`) + "'"
}
