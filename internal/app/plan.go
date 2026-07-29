package app

import "context"

// PlanInstall previews an install-family mutation without writing disk.
func PlanInstall(ctx context.Context, ac *Context, opts InstallOptions) (InstallResult, error) {
	opts.DryRun = true
	return Install(ctx, ac, opts)
}

// PlanUpdate previews an update mutation without writing disk.
func PlanUpdate(ctx context.Context, ac *Context, opts UpdateOptions) (InstallResult, error) {
	opts.Install.DryRun = true
	return Update(ctx, ac, opts)
}
