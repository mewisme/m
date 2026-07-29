package app

import (
	"context"
)

// DedupeOptions controls m dedupe.
type DedupeOptions struct {
	DryRun bool
	InstallOptions
}

// Dedupe re-resolves the lock graph and consolidates duplicate package names when possible.
func Dedupe(ctx context.Context, ac *Context, opts DedupeOptions) (InstallResult, error) {
	inst := opts.InstallOptions
	inst.Dedupe = true
	inst.DryRun = opts.DryRun
	return runInstallTxn(ctx, ac, inst, nil, nil)
}
