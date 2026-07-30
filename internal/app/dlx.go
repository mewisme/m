package app

import (
	"context"
	"io"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/runner/dlx"
	"github.com/mewisme/mew/internal/runner/envexec"
)

// DLXOptions configures mx remote/local execution.
type DLXOptions struct {
	ModeA         bool
	PackageSpecs  []dlx.PackageSpec
	Command       string
	ForwardedArgs []string
	AssumeYes     bool
	Offline       bool
	Interactive   bool
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
}

// MXCachePruneOptions configures mx cache prune.
type MXCachePruneOptions struct {
	DryRun    bool
	OlderThan time.Duration
}

// MXCacheRoot returns the mx cache directory for ac.
func MXCacheRoot(ac *Context) string {
	if ac == nil || ac.Config == nil {
		return ""
	}
	return config.MXCacheDir(ac.Config)
}

// DLX runs mx package execution (local-first, then remote cache path).
func DLX(ctx context.Context, ac *Context, opts DLXOptions) (runner.ExecResult, error) {
	var empty runner.ExecResult
	if ac == nil || ac.Config == nil {
		return empty, apperr.New(apperr.Internal, "app.dlx", "", "missing app context")
	}
	if len(opts.PackageSpecs) == 0 {
		return empty, apperr.New(apperr.Usage, "app.dlx", "", "missing package spec")
	}
	mode := envexec.DLXModeExplicitPackages
	if opts.ModeA {
		mode = envexec.DLXModePackageCommand
	}
	deps := providerDeps(ac)
	deps.DLX.Interactive = func() bool { return opts.Interactive }
	deps.Stdin = opts.Stdin
	deps.Stdout = opts.Stdout
	deps.Stderr = opts.Stderr
	policy := envexec.LockedProviderPolicy(envexec.SourceDLX)
	if opts.Offline {
		policy.Network = envexec.NetworkForbidden
	}
	orch := envexec.Orchestrator{
		Providers: envexec.DefaultProviderRegistry(),
		Leases:    envexec.DLXLeaseManager{MXCacheRoot: MXCacheRoot(ac)},
		Reporter:  ac.Reporter,
	}
	started := time.Now()
	result, err := orch.Execute(ctx, deps, envexec.ExecutionRequest{
		Source: envexec.DLXSource{
			Packages: opts.PackageSpecs,
			Mode:     mode,
			Offline:  opts.Offline,
			Yes:      opts.AssumeYes,
		},
		Command: envexec.CommandRequest{
			Name: opts.Command,
			Args: opts.ForwardedArgs,
		},
		Policy: policy,
		IO: envexec.IOStreams{
			Stdin:  opts.Stdin,
			Stdout: opts.Stdout,
			Stderr: opts.Stderr,
		},
		Suspend: ac.SuspendUI,
		Resume:  ac.ResumeUI,
	})
	name := opts.Command
	if name == "" && len(opts.PackageSpecs) > 0 {
		name = opts.PackageSpecs[0].Name
	}
	emitExecCompletion(ac, name, time.Since(started), result.ExitCode, err)
	return result, err
}

// PruneMXCache prunes stale mx execution environments.
func PruneMXCache(ctx context.Context, ac *Context, opts MXCachePruneOptions) ([]dlx.PruneCandidate, error) {
	_ = ctx
	if ac == nil || ac.Config == nil {
		return nil, apperr.New(apperr.Internal, "app.dlx", "", "missing app context")
	}
	return dlx.PruneEnvironments(dlx.PruneOptions{
		MXCacheDir:    MXCacheRoot(ac),
		RetentionDays: config.Int(ac.Config, "runner.mx.cache.retention_days", 7),
		OlderThan:     opts.OlderThan,
		DryRun:        opts.DryRun,
	})
}
