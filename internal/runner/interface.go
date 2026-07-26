// Package runner builds environments for scripts, exec, and dlx.
package runner

import "context"

// ScriptEnv is a placeholder environment for script execution.
type ScriptEnv struct {
	Dir  string
	Vars []string
}

// ScriptRunner runs package scripts with cancellation.
type ScriptRunner interface {
	Run(ctx context.Context, script string, env ScriptEnv) error
}
