package app

import (
	"context"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

// mutationPrepareFn runs after ReopenProject and before resolve under mutation ownership.
type mutationPrepareFn func(ctx context.Context, ac *Context, proj *project.Project, opts *InstallOptions) error

func prepareAddDependency(ctx context.Context, ac *Context, proj *project.Project, opts *InstallOptions) error {
	if opts == nil || opts.AddSpec == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return apperr.Wrap(apperr.Cancelled, "app.add", opts.AddSpec, err)
	}
	name, verSpec := parsePackageSpec(opts.AddSpec)
	if name == "" {
		return apperr.New(apperr.Usage, "app.add", opts.AddSpec, "invalid package name")
	}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return err
	}
	base := registry.ResolveBaseForPackage(ac.Config, proj.Root, proj.Identity, name)
	pack, err := eng.Client.Packument(ctx, base, name)
	if err != nil {
		return err
	}
	meta, err := pack.SelectVersion(verSpec)
	if err != nil {
		return err
	}
	rng := "^" + meta.Version
	if opts.AddSaveExact {
		rng = meta.Version
	}
	field := "dependencies"
	if opts.AddDev {
		field = "devDependencies"
	}
	if len(opts.Filter) > 0 {
		return prepareFilteredAdd(ctx, ac, proj, opts, name, rng, field)
	}
	return proj.Doc.SetDependency(field, name, rng)
}
