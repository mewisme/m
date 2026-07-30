package app

import (
	"context"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/runner/envexec"
)

// ExecOptions configures m exec and direct bin dispatch.
type ExecOptions struct {
	Command         string
	PackageFilter   string
	ForwardedArgs   []string
	Filters         []string
	Recursive       bool
	RequireVerified bool
	SnapshotID      string
	CapsulePath     string
}

// Exec runs a local package binary in exactly one importer context.
func Exec(ctx context.Context, ac *Context, opts ExecOptions) (runner.ExecResult, error) {
	var empty runner.ExecResult
	if opts.Recursive {
		return empty, apperr.New(apperr.Usage, "app.exec", "", "m exec does not support -r/--recursive")
	}
	cwd := ac.CWD
	if cwd == "" {
		return empty, apperr.New(apperr.Internal, "app.exec", "", "missing cwd")
	}
	var source envexec.SourceRequest
	switch {
	case opts.SnapshotID != "" && opts.CapsulePath != "":
		return empty, apperr.New(apperr.Usage, "app.exec", "", "snapshot and capsule are mutually exclusive")
	case opts.SnapshotID != "":
		root, err := discoverProjectRoot(ac)
		if err != nil {
			return empty, err
		}
		source = envexec.SnapshotSource{ProjectRoot: root, SnapshotID: opts.SnapshotID}
	case opts.CapsulePath != "":
		source = envexec.CapsuleSource{Path: opts.CapsulePath}
	default:
		source = envexec.ProjectSource{CWD: cwd}
	}
	policy := envexec.LockedProviderPolicy(source.Kind())
	if opts.RequireVerified {
		policy.Verification = envexec.VerificationRequired
	}
	return execViaOrchestrator(ctx, ac, envexec.ExecutionRequest{
		Source: source,
		Command: envexec.CommandRequest{
			Name:              opts.Command,
			OwnerDependency:   opts.PackageFilter,
			RequireOwnerMatch: opts.PackageFilter != "",
			Args:              opts.ForwardedArgs,
		},
		Policy:  policy,
		Filters: opts.Filters,
	})
}
