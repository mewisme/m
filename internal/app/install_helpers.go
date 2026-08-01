package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/contentid"
	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lifecycle"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/plan"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/store"
)

type enrichTarget struct {
	pkg  *graph.Package
	base string
}

func resolveForInstall(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions, manifestChanged bool) (*resolver.Resolution, error) {
	if opts.Dedupe {
		return resolveForDedupe(ctx, ac, proj, opts)
	}
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
		PnpmMajor:       resolvePnpmMajor(ac, proj, opts),
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
		PnpmMajor:         resolvePnpmMajor(ac, proj, InstallOptions{}),
	})
}

func resolveForDedupe(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions) (*resolver.Resolution, error) {
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return nil, err
	}
	prior, err := readLockHints(ctx, ac, proj)
	if err != nil {
		return nil, err
	}
	if prior == nil {
		return nil, apperr.New(apperr.Lockfile, "app.dedupe", LockPath(proj), "no lockfile to dedupe")
	}
	ropts := resolver.ResolveOptions{
		Dedupe:         true,
		PriorForDedupe: prior,
		OmitRootDev:    opts.Prod,
		Policy:         resolver.PolicyFromEffective(ac.Config),
		PnpmMajor:      resolvePnpmMajor(ac, proj, opts),
	}
	return eng.ResolveProject(ctx, proj, ropts)
}

func resolvePnpmMajor(ac *Context, proj *project.Project, opts InstallOptions) int {
	if opts.PnpmMajor != 0 {
		return opts.PnpmMajor
	}
	if proj == nil || proj.Identity != project.IdentityPNPM {
		return 0
	}
	prior, err := project.ReadLockfileBytes(proj.Root, proj.Identity)
	if err != nil {
		return 0
	}
	det, err := detectPnpmLock(prior, proj, 0)
	if err != nil {
		return 0
	}
	return det.ProducerMajor
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

func fetchPackages(ctx context.Context, ac *Context, proj *project.Project, g *graph.Graph, ext lockfile.Extensions, extractRoot string, useGlobalStore bool, preExtracts map[string]string, onProgress func(completed int64, current string)) (FetchOutcome, error) {
	if preExtracts == nil {
		preExtracts = map[string]string{}
	}
	if err := enrichRegistryTarballs(ctx, ac, proj, g, preExtracts); err != nil {
		return FetchOutcome{}, err
	}
	if err := fetchSourcePackages(ctx, ac, proj, ext, g, extractRoot, preExtracts); err != nil {
		return FetchOutcome{}, err
	}
	if useGlobalStore {
		return fetchAndImportGraph(ctx, ac, g, preExtracts, onProgress)
	}
	extracts, err := fetchGraphLegacy(ctx, ac, g, extractRoot, preExtracts, onProgress)
	return FetchOutcome{Extracts: extracts}, err
}

func fetchSourcePackages(ctx context.Context, ac *Context, proj *project.Project, ext lockfile.Extensions, g *graph.Graph, extractRoot string, preExtracts map[string]string) error {
	if g == nil || proj == nil {
		return nil
	}
	locals, err := resolver.DecodeLocalSources(ext)
	if err != nil {
		return err
	}
	gits, err := resolver.DecodeGitSources(ext)
	if err != nil {
		return err
	}
	if len(locals) == 0 && len(gits) == 0 {
		return nil
	}
	if preExtracts == nil {
		preExtracts = map[string]string{}
	}
	offline := ac != nil && ac.Config != nil && config.Bool(ac.Config, "offline", false)
	for _, pkg := range g.Packages {
		if err := ctx.Err(); err != nil {
			return err
		}
		key := pkg.ID.Key()
		if preExtracts[key] != "" {
			continue
		}
		if git, ok := gits[key]; ok {
			dest := filepath.Join(extractRoot, sanitizeKeyDir(key))
			if err := fetch.FetchGit(ctx, fetch.GitOptions{
				URL: git.URL, Commit: git.Commit, Dest: dest, Offline: offline,
			}); err != nil {
				return err
			}
			preExtracts[key] = dest
			continue
		}
		loc, ok := locals[key]
		if !ok {
			continue
		}
		switch loc.Protocol {
		case "file", "portal":
			abs := filepath.Join(proj.Root, filepath.FromSlash(loc.Path))
			preExtracts[key] = abs
		case "tarball":
			dest := filepath.Join(extractRoot, sanitizeKeyDir(key))
			if err := fetch.MaterializeLocalTarball(ctx, proj.Root, loc.Path, dest); err != nil {
				return err
			}
			preExtracts[key] = dest
		}
	}
	return nil
}

func enrichRegistryTarballs(ctx context.Context, ac *Context, proj *project.Project, g *graph.Graph, preExtracts map[string]string) error {
	if g == nil || ac == nil || proj == nil {
		return nil
	}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return err
	}
	if eng.Client == nil {
		return nil
	}
	preferOffline := config.Bool(ac.Config, "prefer-offline", false)
	offline := config.Bool(ac.Config, "offline", false)

	byBase := map[string][]enrichTarget{}
	for i := range g.Packages {
		pkg := &g.Packages[i]
		key := pkg.ID.Key()
		if pkg.TarballURL != "" || preExtracts[key] != "" {
			continue
		}
		if pkg.Integrity == "" {
			continue
		}
		base := registry.ResolveBaseForPackage(ac.Config, proj.Root, proj.Identity, pkg.ID.Name)
		if preferOffline || offline {
			if packumentCached(eng.Client, base, pkg.ID.Name) {
				if err := applyPackumentTarball(eng.Client, base, pkg); err != nil {
					return err
				}
				continue
			}
			if offline {
				return apperr.New(apperr.Network, "app.install.fetch", pkg.ID.Key(),
					"packument not in cache (offline mode)")
			}
		}
		byBase[base] = append(byBase[base], enrichTarget{pkg: pkg, base: base})
	}
	for base, targets := range byBase {
		if err := enrichPackumentsBatch(ctx, eng.Client, base, targets); err != nil {
			return err
		}
	}
	return nil
}

func enrichPackumentsBatch(ctx context.Context, client *registry.Client, base string, targets []enrichTarget) error {
	if len(targets) == 0 {
		return nil
	}
	names := make([]string, 0, len(targets))
	seen := map[string]struct{}{}
	for _, t := range targets {
		if _, ok := seen[t.pkg.ID.Name]; ok {
			continue
		}
		seen[t.pkg.ID.Name] = struct{}{}
		names = append(names, t.pkg.ID.Name)
	}
	sort.Strings(names)
	packs, err := client.Packuments(ctx, base, names)
	if err != nil {
		if len(targets) == 1 {
			return apperr.Wrap(apperr.Network, "app.install.fetch", targets[0].pkg.ID.Key(), err)
		}
		return apperr.Wrap(apperr.Network, "app.install.fetch", base, err)
	}
	for _, t := range targets {
		pack, ok := packs[t.pkg.ID.Name]
		if !ok {
			return apperr.New(apperr.Network, "app.install.fetch", t.pkg.ID.Key(),
				fmt.Sprintf("packument %q missing from batch response", t.pkg.ID.Name))
		}
		if err := applyPackumentVersion(t.base, t.pkg, pack); err != nil {
			return err
		}
	}
	return nil
}

func applyPackumentTarball(client *registry.Client, base string, pkg *graph.Package) error {
	if client == nil || pkg == nil {
		return nil
	}
	cache := client.Cache()
	if cache == nil {
		return apperr.New(apperr.Network, "app.install.fetch", pkg.ID.Key(), "packument not in cache")
	}
	body, _, ok := cache.Lookup(registry.OriginKey(base), pkg.ID.Name)
	if !ok {
		return apperr.New(apperr.Network, "app.install.fetch", pkg.ID.Key(), "packument not in cache")
	}
	pack, err := registry.ParsePackument(body)
	if err != nil {
		return apperr.Wrap(apperr.Network, "app.install.fetch", pkg.ID.Key(), err)
	}
	return applyPackumentVersion(base, pkg, pack)
}

func applyPackumentVersion(base string, pkg *graph.Package, pack *registry.Packument) error {
	baseVer := registryBaseVersion(pkg.ID.Version)
	meta, ok := pack.Versions[baseVer]
	if !ok {
		return apperr.New(apperr.Network, "app.install.fetch", pkg.ID.Key(),
			fmt.Sprintf("version %q not in packument", baseVer))
	}
	pkg.TarballURL = registry.AbsoluteTarballURL(base, pkg.ID.Name, meta.Dist.Tarball)
	if pkg.TarballURL == "" {
		pkg.TarballURL = meta.Dist.Tarball
	}
	return nil
}

func registryBaseVersion(version string) string {
	if i := strings.IndexByte(version, '('); i >= 0 {
		return version[:i]
	}
	if i := strings.IndexByte(version, '#'); i >= 0 {
		return version[:i]
	}
	return version
}

// FetchOutcome carries fetch/import results and store cleanup warnings.
type FetchOutcome struct {
	Extracts                 map[string]string
	LinkSummary              *linker.LinkSummary
	Downloaded               int
	Reused                   int
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

func fetchGraphLegacy(ctx context.Context, ac *Context, g *graph.Graph, extractRoot string, preExtracts map[string]string, onProgress func(completed int64, current string)) (map[string]string, error) {
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "app.install", extractRoot, err)
	}
	dl, err := newDownloader(ac)
	if err != nil {
		return nil, err
	}
	extracts := make(map[string]string, len(g.Packages))
	opts := archive.DefaultOptions()
	var completed int64
	total := int64(len(g.Packages))
	for _, pkg := range g.Packages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		key := pkg.ID.Key()
		if dir, ok := preExtracts[key]; ok {
			extracts[key] = dir
			completed++
			if onProgress != nil {
				onProgress(completed, pkg.ID.Name)
			}
			continue
		}
		if strings.TrimSpace(pkg.TarballURL) == "" {
			completed++
			if onProgress != nil {
				onProgress(completed, pkg.ID.Name)
			}
			continue
		}
		if onProgress != nil {
			onProgress(completed, pkg.ID.Name)
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
		completed++
		if onProgress != nil {
			onProgress(completed, pkg.ID.Name)
		}
	}
	// ponytail: final progress tick to ensure spinner reaches total/total.
	if onProgress != nil {
		onProgress(completed, "")
	}
	_ = total
	return extracts, nil
}

func fetchAndImportGraph(ctx context.Context, ac *Context, g *graph.Graph, preExtracts map[string]string, onProgress func(completed int64, current string)) (FetchOutcome, error) {
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
	var completed int64
	for _, pkg := range g.Packages {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		key := pkg.ID.Key()
		if dir, ok := preExtracts[key]; ok {
			out.Extracts[key] = dir
			out.Reused++
			completed++
			if onProgress != nil {
				onProgress(completed, pkg.ID.Name)
			}
			continue
		}
		if strings.TrimSpace(pkg.TarballURL) == "" {
			completed++
			if onProgress != nil {
				onProgress(completed, pkg.ID.Name)
			}
			continue
		}
		if onProgress != nil {
			onProgress(completed, pkg.ID.Name)
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
		out.Downloaded++
		completed++
		if onProgress != nil {
			onProgress(completed, pkg.ID.Name)
		}
	}
	if onProgress != nil {
		onProgress(completed, "")
	}
	out.LinkSummary = &linker.LinkSummary{}
	return out, nil
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

	// Phase 1: exact key matches.
	unmatchedPrior := make(map[string]string, len(prior))
	for k, v := range prior {
		unmatchedPrior[k] = v
	}
	unmatchedNext := make(map[string]string, len(next))

	type keyVersion struct {
		key     string
		version string
	}

	for k, ver := range next {
		old, ok := prior[k]
		if !ok {
			unmatchedNext[k] = ver
			continue
		}
		delete(unmatchedPrior, k)
		if old != ver {
			res.Changed++
			res.PackageChanges = append(res.PackageChanges, PackageChange{
				Kind:        PackageChangeUpdated,
				Name:        packageNameFromKey(k),
				FromVersion: old,
				ToVersion:   ver,
				FromKey:     k,
				ToKey:       k,
			})
		}
	}

	// Phase 2: group unmatched entries by package name and pair deterministically.
	type nameGroup struct {
		prior []keyVersion
		next  []keyVersion
	}
	groups := map[string]*nameGroup{}

	for k, v := range unmatchedPrior {
		n := packageNameFromKey(k)
		if groups[n] == nil {
			groups[n] = &nameGroup{}
		}
		groups[n].prior = append(groups[n].prior, keyVersion{key: k, version: v})
	}
	for k, v := range unmatchedNext {
		n := packageNameFromKey(k)
		if groups[n] == nil {
			groups[n] = &nameGroup{}
		}
		groups[n].next = append(groups[n].next, keyVersion{key: k, version: v})
	}

	names := make([]string, 0, len(groups))
	for n := range groups {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		g := groups[name]
		sort.Slice(g.prior, func(i, j int) bool {
			if g.prior[i].version != g.prior[j].version {
				return g.prior[i].version < g.prior[j].version
			}
			return g.prior[i].key < g.prior[j].key
		})
		sort.Slice(g.next, func(i, j int) bool {
			if g.next[i].version != g.next[j].version {
				return g.next[i].version < g.next[j].version
			}
			return g.next[i].key < g.next[j].key
		})

		pi, ni := 0, 0
		for pi < len(g.prior) && ni < len(g.next) {
			res.Changed++
			res.PackageChanges = append(res.PackageChanges, PackageChange{
				Kind:        PackageChangeUpdated,
				Name:        name,
				FromVersion: g.prior[pi].version,
				ToVersion:   g.next[ni].version,
				FromKey:     g.prior[pi].key,
				ToKey:       g.next[ni].key,
			})
			pi++
			ni++
		}
		for ; pi < len(g.prior); pi++ {
			res.Removed++
			res.PackageChanges = append(res.PackageChanges, PackageChange{
				Kind:        PackageChangeRemoved,
				Name:        name,
				FromVersion: g.prior[pi].version,
				FromKey:     g.prior[pi].key,
			})
		}
		for ; ni < len(g.next); ni++ {
			res.Added++
			res.PackageChanges = append(res.PackageChanges, PackageChange{
				Kind:      PackageChangeAdded,
				Name:      name,
				ToVersion: g.next[ni].version,
				ToKey:     g.next[ni].key,
			})
		}
	}

	// Sort PackageChanges for deterministic output.
	sort.Slice(res.PackageChanges, func(i, j int) bool {
		ci, cj := res.PackageChanges[i], res.PackageChanges[j]
		if ci.Kind != cj.Kind {
			return ci.Kind < cj.Kind
		}
		if ci.Name != cj.Name {
			return ci.Name < cj.Name
		}
		if ci.FromVersion != cj.FromVersion {
			return ci.FromVersion < cj.FromVersion
		}
		if ci.ToVersion != cj.ToVersion {
			return ci.ToVersion < cj.ToVersion
		}
		if ci.FromKey != cj.FromKey {
			return ci.FromKey < cj.FromKey
		}
		return ci.ToKey < cj.ToKey
	})

	return res
}

// MutationPlanInput carries graph and install context for dry-run plans.
type MutationPlanInput struct {
	PriorKeys     map[string]string
	Graph         *graph.Graph
	IgnoreScripts bool
	AC            *Context
}

// BuildMutationPlan builds a plan.Plan for dry-run output.
func BuildMutationPlan(in MutationPlanInput) (*plan.Plan, error) {
	p := &plan.Plan{SchemaVersion: plan.SchemaVersion}
	if in.PriorKeys == nil {
		in.PriorKeys = map[string]string{}
	}
	nextKeys := packageKeysFromGraph(in.Graph)
	for key := range in.PriorKeys {
		if _, ok := nextKeys[key]; !ok {
			p.Operations = append(p.Operations, plan.Operation{
				Op: "unlink", Subject: key,
			})
		}
	}
	scriptsEnabled := !in.IgnoreScripts && in.AC != nil && lifecycle.Enabled(in.AC.Config)
	if in.Graph != nil {
		for _, pkg := range in.Graph.Packages {
			key := pkg.ID.Key()
			p.Desired = append(p.Desired, plan.DesiredState{
				PackageKey: key,
				Integrity:  pkg.Integrity,
			})
			oldVer, had := in.PriorKeys[key]
			if had && oldVer == pkg.ID.Version {
				continue
			}
			if needsFetch(in.AC, pkg) {
				detail := pkg.TarballURL
				if detail == "" {
					detail = pkg.Integrity
				}
				p.Operations = append(p.Operations, plan.Operation{
					Op: "fetch", Subject: key, Detail: detail,
				})
			}
			p.Operations = append(p.Operations, plan.Operation{
				Op: "link", Subject: key, Detail: pkg.ID.Name + "@" + pkg.ID.Version,
			})
			if scriptsEnabled {
				p.Operations = append(p.Operations, plan.Operation{
					Op: "script", Subject: key, Detail: "lifecycle",
				})
			}
		}
	}
	p.Commits = append(p.Commits,
		plan.CommitAction{Op: "write-lock"},
		plan.CommitAction{Op: "publish", Subject: "node_modules"},
	)
	return p, p.Normalize()
}

// directPackageKeys returns the set of package keys that are direct dependencies
// of any importer (workspace package) in the graph. Transitive-only packages are excluded.
func directPackageKeys(g *graph.Graph) map[string]bool {
	if g == nil {
		return nil
	}
	importerIDs := make(map[string]bool, len(g.Importers))
	for _, imp := range g.Importers {
		importerIDs[string(imp.ID)] = true
	}
	// Also treat the root importer as an importer for single-package projects.
	importerIDs[string(graph.RootImporter)] = true

	direct := make(map[string]bool)
	for _, e := range g.Edges {
		if importerIDs[e.From] {
			direct[e.To] = true
		}
	}
	return direct
}

// filterDirectChanges returns only PackageChanges whose ToKey or FromKey is in directKeys.
func filterDirectChanges(changes []PackageChange, directKeys map[string]bool) []PackageChange {
	if len(directKeys) == 0 {
		return nil
	}
	out := make([]PackageChange, 0, len(changes))
	for _, c := range changes {
		if directKeys[c.ToKey] || directKeys[c.FromKey] {
			out = append(out, c)
		}
	}
	return out
}

func needsFetch(ac *Context, pkg graph.Package) bool {
	if strings.TrimSpace(pkg.TarballURL) == "" && strings.TrimSpace(pkg.Integrity) == "" {
		return false
	}
	if ac == nil || ac.Config == nil {
		return true
	}
	if strings.TrimSpace(pkg.Integrity) != "" && packageContentCached(ac, pkg.Integrity) {
		return false
	}
	return strings.TrimSpace(pkg.TarballURL) != "" || strings.TrimSpace(pkg.Integrity) != ""
}

func packageContentCached(ac *Context, integrity string) bool {
	parsed, err := fetch.ParseIntegrity(integrity)
	if err != nil {
		return false
	}
	blobs := store.NewDir(config.BlobCacheDir(ac.Config))
	if blobs.Exists(store.Key(parsed.BlobPath())) {
		return true
	}
	storeRoot, err := config.StoreRoot(ac.Config)
	if err != nil {
		return false
	}
	pkgStore := store.NewPackageStore(storeRoot)
	key, err := store.PackageKeyFromIdentity(parsed.Identity())
	if err != nil {
		return false
	}
	return pkgStore.VerifyPackage(context.Background(), key) == nil
}
