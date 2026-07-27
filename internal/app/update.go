package app

import (
	"context"
	"os"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

// UpdateOptions controls m update.
type UpdateOptions struct {
	Targets []string
	Latest  bool
}

// Update re-resolves with incremental hints and writes m.lock.
func Update(ctx context.Context, ac *Context, opts UpdateOptions) error {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return err
	}
	prior, err := readLockHints(ctx, ac, proj)
	if err != nil {
		return err
	}
	if opts.Latest {
		if err := bumpDependencyRanges(ctx, ac, proj, opts.Targets); err != nil {
			return err
		}
		norm, err := manifest.ToNormalized(proj.Doc)
		if err != nil {
			return err
		}
		proj.Normalized = norm
	}
	eng, err := resolver.NewFromApp(ac.Config, proj, os.Environ())
	if err != nil {
		return err
	}
	ropts := resolver.ResolveOptions{
		Prior:             prior,
		Hints:             prior,
		UpdateTargets:     opts.Targets,
		IncrementalUpdate: true,
	}
	res, err := eng.ResolveProject(ctx, proj, ropts)
	if err != nil {
		return err
	}
	if opts.Latest {
		if err := proj.Doc.Write(""); err != nil {
			return err
		}
	}
	return WriteLock(ctx, ac, res)
}

func bumpDependencyRanges(ctx context.Context, ac *Context, proj *project.Project, targets []string) error {
	eng, err := resolver.NewFromApp(ac.Config, proj, os.Environ())
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
