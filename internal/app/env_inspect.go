package app

import (
	"context"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner/dlx"
	"github.com/mewisme/mew/internal/runner/envexec"
)

// EnvInspectOptions configures m env inspect.
type EnvInspectOptions struct {
	SourceKind  envexec.SourceKind
	CWD         string
	Filters     []string
	Package     string
	SnapshotID  string
	CapsulePath string
	DLXSpecs    []dlx.PackageSpec
	DLXMode     envexec.DLXMode
	Command     string
	Offline     bool
}

// InspectEnvironment returns a plan-only environment report.
func InspectEnvironment(ctx context.Context, ac *Context, opts EnvInspectOptions) (envexec.InspectReport, error) {
	var empty envexec.InspectReport
	if ac == nil || ac.Config == nil {
		return empty, apperr.New(apperr.Internal, "app.env.inspect", "", "missing app context")
	}
	req, err := inspectRequest(ac, opts)
	if err != nil {
		return empty, err
	}
	return envexec.InspectEnvironment(ctx, providerDeps(ac), envexec.DefaultProviderRegistry(), req)
}

// InspectEnvironmentJSON returns the v1 inspect JSON document.
func InspectEnvironmentJSON(ctx context.Context, ac *Context, opts EnvInspectOptions) ([]byte, error) {
	report, err := InspectEnvironment(ctx, ac, opts)
	if err != nil {
		return nil, err
	}
	b, err := envexec.EncodeInspectReport(report)
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "app.env.inspect", "", err)
	}
	return append(b, '\n'), nil
}

func inspectRequest(ac *Context, opts EnvInspectOptions) (envexec.ExecutionRequest, error) {
	var source envexec.SourceRequest
	cwd := opts.CWD
	if cwd == "" {
		cwd = ac.CWD
	}
	switch opts.SourceKind {
	case envexec.SourceProject:
		if cwd == "" {
			root, err := discoverProjectRoot(ac)
			if err != nil {
				return envexec.ExecutionRequest{}, err
			}
			cwd = root
		}
		source = envexec.ProjectSource{CWD: cwd}
	case envexec.SourceDLX:
		source = envexec.DLXSource{Packages: opts.DLXSpecs, Mode: opts.DLXMode, Offline: opts.Offline}
	case envexec.SourceSnapshot:
		root, err := discoverProjectRoot(ac)
		if err != nil {
			return envexec.ExecutionRequest{}, err
		}
		source = envexec.SnapshotSource{ProjectRoot: root, SnapshotID: opts.SnapshotID}
	case envexec.SourceCapsule:
		source = envexec.CapsuleSource{Path: opts.CapsulePath}
	default:
		return envexec.ExecutionRequest{}, apperr.New(apperr.Usage, "app.env.inspect", "", "unsupported inspect source")
	}
	policy := envexec.LockedProviderPolicy(source.Kind())
	return envexec.ExecutionRequest{
		Source:  source,
		Command: envexec.CommandRequest{Name: opts.Command, OwnerDependency: opts.Package},
		Policy:  policy,
		Filters: opts.Filters,
	}, nil
}
