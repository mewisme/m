package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/archive"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/contentid"
	"github.com/mewisme/m/internal/fetch"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/plan"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/store"
)

func resolveForInstall(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions, manifestChanged bool) (*resolver.Resolution, error) {
	if opts.Update != nil {
		return resolveForUpdate(ctx, ac, proj, *opts.Update)
	}
	eng, err := resolver.NewFromApp(ac.Config, proj, os.Environ())
	if err != nil {
		return nil, err
	}
	ropts := resolver.ResolveOptions{
		OmitRootDev: opts.Prod,
		Policy:      resolver.PolicyFromEffective(ac.Config),
	}
	if !manifestChanged {
		if prior, err := readLockHints(ctx, ac, proj); err == nil && prior != nil {
			ropts.Prior = prior
			ropts.Hints = prior
		}
		return eng.Resolve(ctx, proj.Root, ropts)
	}
	norm, err := manifest.ToNormalized(proj.Doc)
	if err != nil {
		return nil, err
	}
	proj.Normalized = norm
	return eng.ResolveProject(ctx, proj, ropts)
}

func resolveForUpdate(ctx context.Context, ac *Context, proj *project.Project, u UpdateResolveOptions) (*resolver.Resolution, error) {
	eng, err := resolver.NewFromApp(ac.Config, proj, os.Environ())
	if err != nil {
		return nil, err
	}
	prior, err := readLockHints(ctx, ac, proj)
	if err != nil {
		return nil, err
	}
	fps, err := readLockFingerprints(proj.Root)
	if err != nil {
		return nil, err
	}
	if proj.Normalized == nil {
		norm, err := manifest.ToNormalized(proj.Doc)
		if err != nil {
			return nil, err
		}
		proj.Normalized = norm
	}
	return eng.ResolveProject(ctx, proj, resolver.ResolveOptions{
		Prior:             prior,
		Hints:             prior,
		UpdateTargets:     u.Targets,
		IncrementalUpdate: true,
		PriorOverrides:    u.PriorOverrides,
		PriorFingerprints: fps,
		Policy:            resolver.PolicyFromEffective(ac.Config),
	})
}

func readLockFingerprints(root string) (*resolver.PriorFingerprints, error) {
	path := LockPath(root)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "app.update", path, err)
	}
	doc, err := readLockDocument(root)
	if err != nil {
		return nil, err
	}
	return &resolver.PriorFingerprints{
		OverridesFingerprint:      doc.Settings.OverridesFingerprint,
		ResolverPolicyFingerprint: doc.Settings.ResolverPolicyFingerprint,
		TargetPlatformFingerprint: doc.Settings.TargetPlatformFingerprint,
	}, nil
}

func readLockHints(ctx context.Context, ac *Context, proj *project.Project) (*graph.Graph, error) {
	path := LockPath(proj.Root)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "app.install", path, err)
	}
	return ReadLockGraph(ctx, ac)
}

func fetchPackages(ctx context.Context, ac *Context, g *graph.Graph, extractRoot string, useGlobalStore bool) (map[string]string, *linker.LinkSummary, error) {
	if useGlobalStore {
		return fetchAndImportGraph(ctx, ac, g)
	}
	extracts, err := fetchGraphLegacy(ctx, ac, g, extractRoot)
	return extracts, nil, err
}

func fetchGraphLegacy(ctx context.Context, ac *Context, g *graph.Graph, extractRoot string) (map[string]string, error) {
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "app.install", extractRoot, err)
	}
	dl, err := newDownloader(ac)
	if err != nil {
		return nil, err
	}
	extracts := make(map[string]string, len(g.Packages))
	opts := archive.DefaultOptions()
	for _, pkg := range g.Packages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		key := pkg.ID.Key()
		dest := filepath.Join(extractRoot, sanitizeKeyDir(key))
		_ = os.RemoveAll(dest)
		art, err := dl.Download(ctx, fetch.DownloadRequest{
			URL:       pkg.TarballURL,
			Integrity: pkg.Integrity,
			AuthToken: config.AuthToken(ac.Config, os.Environ()),
		})
		if err != nil {
			return nil, err
		}
		if err := archive.Extract(ctx, art.BlobPath, dest, opts); err != nil {
			_ = os.RemoveAll(dest)
			return nil, err
		}
		extracts[key] = dest
	}
	return extracts, nil
}

func fetchAndImportGraph(ctx context.Context, ac *Context, g *graph.Graph) (map[string]string, *linker.LinkSummary, error) {
	dl, err := newDownloader(ac)
	if err != nil {
		return nil, nil, err
	}
	storeRoot, err := config.StoreRoot(ac.Config)
	if err != nil {
		return nil, nil, err
	}
	pkgStore := store.NewPackageStore(storeRoot)
	extracts := make(map[string]string, len(g.Packages))
	for _, pkg := range g.Packages {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		key := pkg.ID.Key()
		art, err := dl.Download(ctx, fetch.DownloadRequest{
			URL:       pkg.TarballURL,
			Integrity: pkg.Integrity,
			AuthToken: config.AuthToken(ac.Config, os.Environ()),
		})
		if err != nil {
			return nil, nil, err
		}
		result, err := pkgStore.ImportFromTarball(ctx, art.BlobPath, contentid.Identity{
			Algo: art.Integrity.Algo,
			Hex:  art.Integrity.Hex,
		})
		if err != nil {
			return nil, nil, err
		}
		pkgKey := result.Key
		if err := pkgStore.VerifyPackage(ctx, pkgKey); err != nil {
			return nil, nil, err
		}
		extracts[key] = pkgStore.PackagePath(pkgKey)
	}
	return extracts, &linker.LinkSummary{}, nil
}

func sanitizeKeyDir(key string) string {
	return strings.NewReplacer("@", "_at_", "/", "_", "#", "_").Replace(key)
}

func priorPackageKeys(ctx context.Context, ac *Context, proj *project.Project) (map[string]string, error) {
	g, err := readLockHints(ctx, ac, proj)
	if err != nil || g == nil {
		return map[string]string{}, nil
	}
	return packageKeysFromGraph(g), nil
}

func packageKeysFromGraph(g *graph.Graph) map[string]string {
	if g == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(g.Packages))
	for _, p := range g.Packages {
		out[p.ID.Key()] = p.ID.Version
	}
	return out
}

func diffKeys(prior, next map[string]string) InstallResult {
	var res InstallResult
	res.Packages = len(next)
	for k, ver := range next {
		old, ok := prior[k]
		if !ok {
			res.Added++
		} else if old != ver {
			res.Changed++
		}
	}
	for k := range prior {
		if _, ok := next[k]; !ok {
			res.Removed++
		}
	}
	return res
}

// BuildMutationPlan builds a plan.Plan for dry-run output.
func BuildMutationPlan(g *graph.Graph) (*plan.Plan, error) {
	p := &plan.Plan{SchemaVersion: plan.SchemaVersion}
	if g == nil {
		return p, p.Normalize()
	}
	for _, pkg := range g.Packages {
		p.Desired = append(p.Desired, plan.DesiredState{
			PackageKey: pkg.ID.Key(),
			Integrity:  pkg.Integrity,
		})
		p.Operations = append(p.Operations, plan.Operation{
			Op: "link", Subject: pkg.ID.Key(), Detail: pkg.ID.Name + "@" + pkg.ID.Version,
		})
	}
	p.Commits = append(p.Commits, plan.CommitAction{Op: "write-lock"}, plan.CommitAction{Op: "publish", Subject: "node_modules"})
	return p, p.Normalize()
}
