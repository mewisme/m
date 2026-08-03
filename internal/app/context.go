package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/prompt"
)

// TxnOutcome records the outcome of a mutation transaction for error reporting.
type TxnOutcome struct {
	RolledBack       bool
	RecoveryRequired bool
}

// Context is the process-level application state for one CLI invocation.
type Context struct {
	CWD            string
	Config         *config.Effective
	ConfigLoadSpec config.LoadSpec
	Reporter       diagnostics.Reporter
	Version        string
	Commit         string
	BuildDate      string
	BinaryName     string // "m", "mew", "mx", "mewx"
	Ctx            context.Context
	// SuspendUI / ResumeUI pause live progress around child I/O (presentation-owned).
	SuspendUI func(context.Context) error
	ResumeUI  func(context.Context) error
	// Prompter is the presentation-selected prompt adapter (may be nil).
	Prompter prompt.Prompter
	// CanPrompt is true when InteractivePolicy permits a prompt this invocation.
	CanPrompt bool
	// TxnOutcome records the transaction outcome after a mutation attempt.
	TxnOutcome *TxnOutcome
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
	BinaryName    string // "m", "mew", "mx", "mewx"
	// Snapshot reuses an already-loaded configuration instead of loading a
	// second time. The CLI needs ui.theme before it can build the reporter
	// this Context carries, so it loads once via LoadConfig and passes the
	// result back here. Nil means New loads its own.
	Snapshot *ConfigSnapshot
}

// ConfigSnapshot is the resolved configuration for one invocation, along with
// the inputs that produced it.
type ConfigSnapshot struct {
	CWD    string
	Config *config.Effective
	Spec   config.LoadSpec
}

type ctxKey struct{}

// LoadConfig resolves the working directory and loads configuration. It is the
// config half of New, split out so a caller that needs a config value in order
// to construct the reporter can load once and hand the result to New.
func LoadConfig(ctx context.Context, opts Options) (ConfigSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cwd := opts.CWD
	if cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ConfigSnapshot{}, err
		}
		cwd = wd
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return ConfigSnapshot{}, err
	}
	cwd = abs

	cliOverlay := map[string]any{}
	if opts.Offline {
		cliOverlay["offline"] = true
	}
	if opts.PreferOffline {
		cliOverlay["prefer_offline"] = true
	}

	env := opts.Env
	if env == nil {
		// intentional: snapshot host env once when caller does not supply Options.Env.
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
			return ConfigSnapshot{}, err
		}
		within, err := config.IsPathWithin(projectRoot, cfgAbs)
		if err != nil {
			return ConfigSnapshot{}, err
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
		return ConfigSnapshot{}, err
	}
	return ConfigSnapshot{CWD: cwd, Config: eff, Spec: spec}, nil
}

// New builds an application context: resolve CWD, load config, attach reporter.
func New(ctx context.Context, opts Options) (*Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot := opts.Snapshot
	if snapshot == nil {
		loaded, err := LoadConfig(ctx, opts)
		if err != nil {
			return nil, err
		}
		snapshot = &loaded
	}

	rep := opts.Reporter
	if rep == nil {
		rep = diagnostics.NewReporter(diagnostics.Options{})
	}

	return &Context{
		CWD:            snapshot.CWD,
		Config:         snapshot.Config,
		ConfigLoadSpec: snapshot.Spec,
		Reporter:       rep,
		Version:        opts.Version,
		Commit:         opts.Commit,
		BuildDate:      opts.BuildDate,
		BinaryName:     opts.BinaryName,
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
