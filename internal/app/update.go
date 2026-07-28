package app

import (
	"context"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
)

// UpdateOptions controls m update.
type UpdateOptions struct {
	Targets []string
	Latest  bool
	Install InstallOptions
}

// Update re-resolves with incremental hints and commits via install transaction.
func Update(ctx context.Context, ac *Context, opts UpdateOptions) (InstallResult, error) {
	if len(opts.Install.Filter) > 0 {
		return InstallResult{}, apperr.New(apperr.Usage, "app.update", "--filter", "filtered update is not supported yet")
	}
	updateParams := &UpdateResolveOptions{Targets: opts.Targets}
	edit := func(proj *project.Project) error {
		norm, err := manifest.ToNormalized(proj.Doc)
		if err != nil {
			return err
		}
		updateParams.PriorOverrides = cloneOverrides(norm.Overrides)
		if doc, err := readLockDocument(proj.Root, proj.Identity); err == nil && doc != nil {
			updateParams.PriorFingerprints = &resolver.PriorFingerprints{
				OverridesFingerprint:      doc.Settings.OverridesFingerprint,
				ResolverPolicyFingerprint: doc.Settings.ResolverPolicyFingerprint,
				TargetPlatformFingerprint: doc.Settings.TargetPlatformFingerprint,
			}
		}
		if opts.Latest {
			if err := bumpDependencyRanges(ctx, ac, proj, opts.Targets); err != nil {
				return err
			}
			norm, err = manifest.ToNormalized(proj.Doc)
			if err != nil {
				return err
			}
		}
		proj.Normalized = norm
		return nil
	}
	inst := opts.Install
	inst.WriteManifest = opts.Latest
	inst.Update = updateParams
	return runInstallTxn(ctx, ac, inst, edit, nil)
}

func bumpDependencyRanges(ctx context.Context, ac *Context, proj *project.Project, targets []string) error {
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return err
	}
	want := map[string]struct{}{}
	if len(targets) == 0 {
		for _, field := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
			for name := range depsForField(proj.Doc, field) {
				want[name] = struct{}{}
			}
		}
	} else {
		for _, t := range targets {
			want[t] = struct{}{}
		}
	}
	for field, deps := range map[string]map[string]string{
		"dependencies":         proj.Doc.Dependencies,
		"devDependencies":      proj.Doc.DevDependencies,
		"optionalDependencies": proj.Doc.OptionalDependencies,
	} {
		for name := range deps {
			if _, ok := want[name]; !ok {
				continue
			}
			base := registry.ResolveBaseForPackage(ac.Config, proj.Root, proj.Identity, name)
			pack, err := eng.Client.Packument(ctx, base, name)
			if err != nil {
				return err
			}
			meta, err := pack.SelectVersion("latest")
			if err != nil {
				return apperr.Wrap(apperr.Resolve, "app.update", name, err)
			}
			if err := proj.Doc.SetDependency(field, name, "^"+meta.Version); err != nil {
				return err
			}
		}
	}
	return nil
}

func depsForField(doc *manifest.Document, field string) map[string]string {
	if doc == nil {
		return nil
	}
	switch field {
	case "dependencies":
		return doc.Dependencies
	case "devDependencies":
		return doc.DevDependencies
	case "optionalDependencies":
		return doc.OptionalDependencies
	default:
		return nil
	}
}

func cloneOverrides(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
