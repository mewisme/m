package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/binresolve"
	"github.com/mewisme/mew/internal/process"
)

// ExecOptions configures local package binary execution.
type ExecOptions struct {
	ProjectRoot     string
	PackageDir      string
	ImporterRel     string
	NodeModules     string
	Command         string
	PackageFilter   string
	ForwardedArgs   []string
	HostEnv         []string
	RequireVerified bool
	AllowUnowned    bool
	GenerationID    string
	Fingerprint     string
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
	// Suspend / Resume pause presentation around child Start/Wait (optional).
	Suspend func(context.Context) error
	Resume  func(context.Context) error
}

// ExecResult summarizes binary execution.
type ExecResult struct {
	Candidate    binmeta.BinCandidate
	UsedFallback bool
	ExitCode     int
}

// Exec resolves, validates, launches, and waits on a local package binary.
func Exec(ctx context.Context, opts ExecOptions, sup process.ProcessSupervisor) (ExecResult, error) {
	var empty ExecResult
	if sup == nil {
		sup = process.NewExecSupervisor()
	}
	resolveOpts := binresolve.Options{
		ProjectRoot:     opts.ProjectRoot,
		ImporterRel:     opts.ImporterRel,
		PackageDir:      opts.PackageDir,
		Command:         opts.Command,
		PackageFilter:   opts.PackageFilter,
		RequireVerified: opts.RequireVerified,
		AllowUnowned:    opts.AllowUnowned,
		GenerationID:    opts.GenerationID,
		Fingerprint:     opts.Fingerprint,
	}
	found, err := binresolve.Resolve(resolveOpts)
	if err != nil {
		return empty, err
	}
	nm := opts.NodeModules
	if nm == "" {
		nm = filepath.Join(opts.PackageDir, "node_modules")
	}
	if err := binresolve.ValidateCandidate(found, nm); err != nil {
		return empty, err
	}
	hostEnv := opts.HostEnv
	if len(hostEnv) == 0 {
		hostEnv = os.Environ()
	}
	env := BuildEnv(EnvOptions{
		HostEnv:     hostEnv,
		InitCWD:     opts.ProjectRoot,
		PackageDir:  opts.PackageDir,
		NodeModules: nm,
		PackageJSON: PackageJSONPath(opts.PackageDir),
		Event:       "exec",
		Script:      opts.Command,
	})
	launch, err := binresolve.BuildLaunchSpec(found, opts.ForwardedArgs, hostEnv, env.Dir)
	if err != nil {
		return empty, err
	}
	path, args := binresolve.LaunchToProcessSpec(launch)
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
	spec := process.Spec{
		Path:   path,
		Args:   args,
		Dir:    launch.Dir,
		Env:    env.Vars,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	if opts.Suspend != nil {
		_ = opts.Suspend(ctx)
	}
	if opts.Resume != nil {
		defer func() { _ = opts.Resume(ctx) }()
	}
	h, err := sup.Start(ctx, spec)
	if err != nil {
		return empty, apperr.Wrap(apperr.Exec, "runner.exec", opts.Command, err)
	}
	waitErr := sup.Wait(ctx, h)
	if errors.Is(ctx.Err(), context.Canceled) {
		return ExecResult{Candidate: found, ExitCode: process.ExitCode(waitErr)},
			apperr.Wrap(apperr.Cancelled, "runner.exec", opts.Command, context.Canceled)
	}
	code := process.ExitCode(waitErr)
	result := ExecResult{Candidate: found, ExitCode: code}
	if code != 0 {
		return result, &apperr.ExitStatus{Code: code, Err: waitErr}
	}
	if waitErr != nil {
		return result, apperr.Wrap(apperr.IO, "runner.exec", opts.Command, waitErr)
	}
	return result, nil
}
