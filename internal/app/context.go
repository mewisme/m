package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/diagnostics"
	"github.com/mewisme/m/internal/project"
)

// Context is the process-level application state for one CLI invocation.
type Context struct {
	CWD            string
	Config         *config.Effective
	ConfigLoadSpec config.LoadSpec
	Reporter       diagnostics.Reporter
	Version        string
	Commit         string
	BuildDate      string
	Ctx            context.Context
}

// Options controls Context construction from CLI globals.
type Options struct {
	CWD           string
	ConfigPath    string
	Offline       bool
	PreferOffline bool
	Env           []string
	Reporter      diagnostics.Reporter
	Version       string
	Commit        string
	BuildDate     string
}

type ctxKey struct{}

// New builds an application context: resolve CWD, load config, attach reporter.
func New(ctx context.Context, opts Options) (*Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cwd := opts.CWD
	if cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		cwd = wd
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	cwd = abs

	cliOverlay := map[string]any{}
	if opts.Offline {
		cliOverlay["offline"] = true
	}
	if opts.PreferOffline {
		cliOverlay["prefer-offline"] = true
	}

	env := opts.Env
	if env == nil {
		env = os.Environ()
	}

	projectRoot := cwd
	if r, err := project.FindRoot(cwd); err == nil {
		projectRoot = r
	}

	snap := config.NewEnvSnapshot(env, runtime.GOOS)

	loadOpts := config.LoadOptions{
		CWD:         cwd,
		ProjectRoot: projectRoot,
		Env:         env,
		EnvSnapshot: snap,
		CLI:         cliOverlay,
		GlobalPath:  config.GlobalConfigPathFromEnv(snap),
	}
	if opts.ConfigPath != "" {
		cfgAbs, err := config.ResolveConfigPath(cwd, opts.ConfigPath)
		if err != nil {
			return nil, err
		}
		within, err := config.IsPathWithin(projectRoot, cfgAbs)
		if err != nil {
			return nil, err
		}
		if within {
			loadOpts.ProjectPath = cfgAbs
			loadOpts.RequireProjectConfig = true
		} else {
			loadOpts.GlobalPath = cfgAbs
			loadOpts.RequireGlobalConfig = true
		}
	}

	spec := config.LoadSpecFromOptions(loadOpts)
	eff, err := config.Load(ctx, spec.LoadOptions())
	if err != nil {
		return nil, err
	}

	rep := opts.Reporter
	if rep == nil {
		rep = diagnostics.NewReporter(diagnostics.Options{})
	}

	return &Context{
		CWD:            cwd,
		Config:         eff,
		ConfigLoadSpec: spec,
		Reporter:       rep,
		Version:        opts.Version,
		Commit:         opts.Commit,
		BuildDate:      opts.BuildDate,
		Ctx:            ctx,
	}, nil
}

// WithContext stores app Context on a Go context.
func WithContext(ctx context.Context, ac *Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, ac)
}

// FromContext retrieves app Context, or nil.
func FromContext(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	ac, _ := ctx.Value(ctxKey{}).(*Context)
	return ac
}
