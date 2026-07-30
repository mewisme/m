// Package runner builds environments for scripts, exec, and dlx.
package runner

import (
	"context"
	"io"

	"github.com/mewisme/mew/internal/diagnostics"
)

// HookStage is one lifecycle hook to execute in order.
type HookStage struct {
	Event  string // npm_lifecycle_event value (e.g. predev, dev, postdev)
	Script string // script body from package.json
}

// ScriptPlan is the hook expansion for one primary script name.
type ScriptPlan struct {
	Name   string
	Stages []HookStage
}

// ScriptEnv is the working directory and environment for one hook stage.
type ScriptEnv struct {
	Dir  string
	Vars []string
}

// RunOptions configures package script execution.
type RunOptions struct {
	ProjectRoot   string
	PackageDir    string
	NodeModules   string
	PackageJSON   string
	PackageName   string
	PackageVer    string
	Scripts       map[string]string
	Selector      string
	IfPresent     bool
	ForwardedArgs []string
	HostEnv       []string
	Reporter      diagnostics.Reporter
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
}

// RunResult summarizes script execution outcome.
type RunResult struct {
	ExitCode int
	Plans    []ScriptPlan
}

// ScriptRunner executes package scripts with cancellation and exit-code propagation.
// Hook stages and regex matches run sequentially; execution stops on the first failure.
type ScriptRunner interface {
	Run(ctx context.Context, opts RunOptions) (RunResult, error)
}
