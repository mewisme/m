package app

import (
	"context"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/workspace"
)

// RunOptions configures package script execution from the CLI.
type RunOptions struct {
	Selector      string
	IfPresent     bool
	ForwardedArgs []string
	Recursive     bool
	Filters       []string
	Concurrency   int
	Order         runner.WorkspaceOrder
	Output        runner.WorkspaceOutputMode
	Bail          bool
	// WorkspaceOnlySet is true when any workspace-only flag was passed.
	WorkspaceOnlySet bool
}

// Run resolves and executes a package.json script in the current project.
func Run(ctx context.Context, ac *Context, opts RunOptions) (runner.RunResult, error) {
	var empty runner.RunResult
	if ac == nil {
		return empty, apperr.New(apperr.Internal, "app.run", "", "missing app context")
	}

	workspaceMode := opts.Recursive || len(opts.Filters) > 0
	if opts.WorkspaceOnlySet && !workspaceMode {
		return empty, apperr.New(apperr.Usage, "app.run", "", "workspace flags require -r or --filter")
	}

	if workspaceMode {
		if ac.Config == nil || !workspace.Enabled(ac.Config) {
			return empty, apperr.New(apperr.Usage, "app.run", "", "workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
		}
		_, err := runWorkspace(ctx, ac, opts)
		return empty, err
	}

	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return empty, err
	}

	packageDir := proj.Root
	if proj.Rel != "." {
		packageDir = filepath.Join(proj.Root, filepath.FromSlash(proj.Rel))
	}

	scripts := map[string]string{}
	pkgName := ""
	pkgVer := ""
	if proj.Doc != nil {
		if proj.Doc.Scripts != nil {
			scripts = proj.Doc.Scripts
		}
		pkgName = proj.Doc.Name
		pkgVer = proj.Doc.Version
	}

	hostEnv := []string(nil)
	if ac.Config != nil {
		hostEnv = ac.Config.Env.Environ()
	}

	emitProjectExecPrep(ac, opts.Selector, pkgName)

	started := time.Now()
	result, err := runner.NewDefaultRunner().Run(ctx, runner.RunOptions{
		ProjectRoot:   ac.CWD,
		PackageDir:    packageDir,
		NodeModules:   filepath.Join(packageDir, "node_modules"),
		PackageName:   pkgName,
		PackageVer:    pkgVer,
		Scripts:       scripts,
		Selector:      opts.Selector,
		IfPresent:     opts.IfPresent,
		ForwardedArgs: opts.ForwardedArgs,
		HostEnv:       hostEnv,
		Reporter:      ac.Reporter,
		Suspend:       ac.SuspendUI,
		Resume:        ac.ResumeUI,
	})
	emitExecCompletion(ac, opts.Selector, time.Since(started), result.ExitCode, err)
	return result, err
}

func runWorkspace(ctx context.Context, ac *Context, opts RunOptions) (runner.WorkspaceResult, error) {
	var empty runner.WorkspaceResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return empty, err
	}
	wg, err := workspace.BuildGraph(proj.Root)
	if err != nil {
		return empty, err
	}
	paths, err := runner.SelectMembers(wg, opts.Recursive, opts.Filters)
	if err != nil {
		return empty, err
	}
	order := opts.Order
	if order == "" {
		order = runner.OrderTopological
	}
	output := opts.Output
	if output == "" {
		output = runner.OutputStream
	}
	hostEnv := []string(nil)
	if ac.Config != nil {
		hostEnv = ac.Config.Env.Environ()
	}
	wsOpts := runner.WorkspaceRunOptions{
		ProjectRoot:   proj.Root,
		Selector:      opts.Selector,
		IfPresent:     opts.IfPresent,
		ForwardedArgs: opts.ForwardedArgs,
		Recursive:     opts.Recursive,
		Filters:       opts.Filters,
		Concurrency:   opts.Concurrency,
		Order:         order,
		Output:        output,
		Bail:          opts.Bail,
		HostEnv:       hostEnv,
	}
	return runner.RunWorkspace(ctx, wg, paths, wsOpts, ac.Reporter)
}
