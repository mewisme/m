package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/contentid"
	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/plan"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/store"
)

func resolveForInstall(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions, manifestChanged bool) (*resolver.Resolution, error) {
	if opts.Update != nil {
		return resolveForUpdate(ctx, ac, proj, *opts.Update)
	}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return nil, err
	}
	ropts := resolver.ResolveOptions{
		OmitRootDev:     opts.Prod,
		Policy:          resolver.PolicyFromEffective(ac.Config),
		Recursive:       opts.Recursive,
		Filter:          append([]string(nil), opts.Filter...),
		MemberManifests: opts.MemberEdits,
	}
	if prior, err := readLockHints(ctx, ac, proj); err == nil && prior != nil {
		ropts.Hints = prior
		if !manifestChanged {
			ropts.Prior = prior
		}
	}
	if !manifestChanged {
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
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return nil, err
	}
	prior, err := readLockHints(ctx, ac, proj)
	if err != nil {
		return nil, err
	}
	fps, err := readLockFingerprints(proj.Root, proj.Identity)
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

func readLockFingerprints(root string, id project.Identity) (*resolver.PriorFingerprints, error) {
	if id != project.IdentityMew {
		return nil, nil
	}
	path := filepath.Join(root, "m.lock")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "app.update", path, err)
	}
	doc, err := readLockDocument(root, id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, nil
	}
	return &resolver.PriorFingerprints{
		OverridesFingerprint:      doc.Settings.OverridesFingerprint,
		ResolverPolicyFingerprint: doc.Settings.ResolverPolicyFingerprint,
		TargetPlatformFingerprint: doc.Settings.TargetPlatformFingerprint,
	}, nil
}

func readLockHints(ctx context.Context, ac *Context, proj *project.Project) (*graph.Graph, error) {
	path := LockPath(proj)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "app.install", path, err)
	}
	return lockfile.ReadGraph(ctx, proj.Root, proj.Identity)
}

func fetchPackages(ctx context.Context, ac *Context, g *graph.Graph, extractRoot string, useGlobalStore bool, preExtracts map[string]string) (FetchOutcome, error) {
	if useGlobalStore {
		return fetchAndImportGraph(ctx, ac, g, preExtracts)
	}
	extracts, err := fetchGraphLegacy(ctx, ac, g, extractRoot, preExtracts)
	return FetchOutcome{Extracts: extracts}, err
}

// FetchOutcome carries fetch/import results and store cleanup warnings.
type FetchOutcome struct {
	Extracts                 map[string]string
	LinkSummary              *linker.LinkSummary
	CleanupWarningCodes      []string
	CleanupWarnings          []string
	StoreCleanupIncomplete   bool
	StoreMaintenanceRequired bool
}

func mergeFetchImportResult(out *FetchOutcome, result store.ImportResult) {
	if out == nil {
		return
	}
	for i, code := range result.CleanupWarningCodes {
		msg := ""
		if i < len(result.CleanupWarnings) {
			msg = result.CleanupWarnings[i]
		}
		if cleanupPairContains(out.CleanupWarningCodes, out.CleanupWarnings, code, msg) {
			continue
		}
		out.CleanupWarningCodes = append(out.CleanupWarningCodes, code)
		out.CleanupWarnings = append(out.CleanupWarnings, msg)
	}
	for i := len(result.CleanupWarningCodes); i < len(result.CleanupWarnings); i++ {
		msg := result.CleanupWarnings[i]
		if cleanupPairContains(out.CleanupWarningCodes, out.CleanupWarnings, "", msg) {
			continue
		}
		out.CleanupWarnings = append(out.CleanupWarnings, msg)
	}
	if len(result.CleanupWarningCodes) > 0 || len(result.CleanupWarnings) > 0 {
		out.StoreCleanupIncomplete = true
		out.StoreMaintenanceRequired = true
	}
}

func fetchGraphLegacy(ctx context.Context, ac *Context, g *graph.Graph, extractRoot string, preExtracts map[string]string) (map[string]string, error) {
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
		if dir, ok := preExtracts[key]; ok {
			extracts[key] = dir
			continue
		}
		dest := filepath.Join(extractRoot, sanitizeKeyDir(key))
		_ = os.RemoveAll(dest)
		art, err := dl.Download(ctx, fetch.DownloadRequest{
			URL:       pkg.TarballURL,
			Integrity: pkg.Integrity,
			AuthToken: config.AuthToken(ac.Config),
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

func fetchAndImportGraph(ctx context.Context, ac *Context, g *graph.Graph, preExtracts map[string]string) (FetchOutcome, error) {
	var out FetchOutcome
	dl, err := newDownloader(ac)
	if err != nil {
		return out, err
	}
	storeRoot, err := config.StoreRoot(ac.Config)
	if err != nil {
		return out, err
	}
	pkgStore := store.NewPackageStore(storeRoot)
	if ac != nil {
		pkgStore.Reporter = ac.Reporter
	}
	out.Extracts = make(map[string]string, len(g.Packages))
	for _, pkg := range g.Packages {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		key := pkg.ID.Key()
		if dir, ok := preExtracts[key]; ok {
			out.Extracts[key] = dir
			continue
		}
		art, err := dl.Download(ctx, fetch.DownloadRequest{
			URL:       pkg.TarballURL,
			Integrity: pkg.Integrity,
			AuthToken: config.AuthToken(ac.Config),
		})
		if err != nil {
			return out, err
		}
		result, err := pkgStore.ImportFromTarball(ctx, art.BlobPath, contentid.Identity{
			Algo: art.Integrity.Algo,
			Hex:  art.Integrity.Hex,
		})
		mergeFetchImportResult(&out, result)
		if err != nil {
			return out, err
		}
		pkgKey := result.Key
		if err := pkgStore.VerifyPackage(ctx, pkgKey); err != nil {
			return out, err
		}
		out.Extracts[key] = pkgStore.PackagePath(pkgKey)
	}
	out.LinkSummary = &linker.LinkSummary{}
	return out, nil
}

func sanitizeKeyDir(key string) string {
	return strings.NewReplacer("@", "_at_", "/", "_", "#", "_").Replace(key)
}

func applyPatchesToExtracts(ctx context.Context, ext lockfile.Extensions, extracts map[string]string) error {
	patches, err := resolver.DecodePatchSources(ext)
	if err != nil {
		return err
	}
	for pkgKey, patch := range patches {
		if err := ctx.Err(); err != nil {
			return err
		}
		dir, ok := extracts[pkgKey]
		if !ok || dir == "" || patch.Path == "" {
			continue
		}
		if err := archive.ApplyUnifiedPatch(ctx, dir, patch.Path); err != nil {
			return err
		}
	}
	return nil
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
