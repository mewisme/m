package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/archive"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/fetch"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker/hoisted"
	"github.com/mewisme/m/internal/plan"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

// InstallOptions controls m install / ci.
type InstallOptions struct {
	Prod   bool
	Frozen bool
	DryRun bool
}

// InstallResult summarizes package changes.
type InstallResult struct {
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Changed  int `json:"changed"`
	Packages int `json:"packages"`
}

// AddOptions controls m add.
type AddOptions struct {
	Dev       bool
	SaveExact bool
	Install   InstallOptions
}

// Install resolves, fetches, links, and publishes node_modules.
func Install(ctx context.Context, ac *Context, opts InstallOptions) (InstallResult, error) {
	var res InstallResult
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.install", "", "missing app context")
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	if opts.Frozen {
		if err := ValidateFrozenLock(ctx, ac); err != nil {
			return res, err
		}
	}

	priorKeys, _ := priorPackageKeys(ctx, ac, proj)
	resolution, err := resolveForInstall(ctx, ac, proj, opts)
	if err != nil {
		return res, err
	}
	res = diffKeys(priorKeys, packageKeysFromGraph(resolution.Graph))

	if opts.DryRun {
		return res, nil
	}

	stageParent := filepath.Join(proj.Root, ".mew-stage")
	if err := os.MkdirAll(stageParent, 0o755); err != nil {
		return res, apperr.Wrap(apperr.IO, "app.install", stageParent, err)
	}
	stageRoot, err := os.MkdirTemp(stageParent, "install-*")
	if err != nil {
		return res, apperr.Wrap(apperr.IO, "app.install", "stage", err)
	}
	defer func() { _ = os.RemoveAll(stageRoot) }()

	extractDir := filepath.Join(stageRoot, "extract")
	stageNM := filepath.Join(stageRoot, "node_modules")
	extracts, err := fetchGraph(ctx, ac, resolution.Graph, extractDir)
	if err != nil {
		return res, err
	}

	h := &hoisted.Linker{NodeModules: stageNM, ExtractDirs: extracts}
	linkPlan, err := h.Plan(ctx, resolution.Graph)
	if err != nil {
		return res, err
	}
	if err := h.Apply(ctx, linkPlan); err != nil {
		return res, err
	}
	if err := WriteLock(ctx, ac, resolution); err != nil {
		return res, err
	}
	liveNM := filepath.Join(proj.Root, "node_modules")
	// ponytail: rename swap only; 0017 adds journaled transaction commit.
	if err := publishNodeModules(stageNM, liveNM); err != nil {
		return res, err
	}
	return res, nil
}

// Add declares a dependency, writes package.json, and installs.
func Add(ctx context.Context, ac *Context, spec string, opts AddOptions) (InstallResult, error) {
	var res InstallResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	name, verSpec := parsePackageSpec(spec)
	if name == "" {
		return res, apperr.New(apperr.Usage, "app.add", spec, "invalid package name")
	}
	eng, err := resolver.NewFromApp(ac.Config, proj, os.Environ())
	if err != nil {
		return res, err
	}
	base := registry.ResolveBaseForPackage(ac.Config, proj.Root, proj.Identity, name)
	pack, err := eng.Client.Packument(ctx, base, name)
	if err != nil {
		return res, err
	}
	meta, err := pack.SelectVersion(verSpec)
	if err != nil {
		return res, err
	}
	rng := "^" + meta.Version
	if opts.SaveExact {
		rng = meta.Version
	}
	field := "dependencies"
	if opts.Dev {
		field = "devDependencies"
	}
	if err := proj.Doc.SetDependency(field, name, rng); err != nil {
		return res, err
	}
	if err := proj.Doc.Write(""); err != nil {
		return res, err
	}
	return Install(ctx, ac, opts.Install)
}

// Remove deletes a dependency from package.json and reinstalls.
func Remove(ctx context.Context, ac *Context, name string, opts InstallOptions) (InstallResult, error) {
	var res InstallResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	fields := []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}
	var removed bool
	for _, field := range fields {
		if err := proj.Doc.RemoveDependency(field, name); err != nil {
			if apperr.CodeOf(err) == apperr.NotFound {
				continue
			}
			return res, err
		}
		removed = true
		break
	}
	if !removed {
		return res, apperr.New(apperr.NotFound, "app.remove", name, "dependency not found")
	}
	if err := proj.Doc.Write(""); err != nil {
		return res, err
	}
	return Install(ctx, ac, opts)
}

func resolveForInstall(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions) (*resolver.Resolution, error) {
	eng, err := resolver.NewFromApp(ac.Config, proj, os.Environ())
	if err != nil {
		return nil, err
	}
	ropts := resolver.ResolveOptions{OmitRootDev: opts.Prod}
	if hints, err := readLockHints(ctx, ac, proj); err == nil && hints != nil {
		ropts.Hints = hints
	}
	return eng.Resolve(ctx, proj.Root, ropts)
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

func fetchGraph(ctx context.Context, ac *Context, g *graph.Graph, extractRoot string) (map[string]string, error) {
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

func sanitizeKeyDir(key string) string {
	return strings.NewReplacer("@", "_at_", "/", "_", "#", "_").Replace(key)
}

// ponytail: rename swap; upgrade to internal/transaction commit in 0017.
func publishNodeModules(stageNM, liveNM string) error {
	backup := liveNM + ".mew-old"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(liveNM); err == nil {
		if err := os.Rename(liveNM, backup); err != nil {
			return apperr.Wrap(apperr.IO, "app.install", liveNM, err)
		}
	}
	if err := os.Rename(stageNM, liveNM); err != nil {
		if _, statErr := os.Stat(backup); statErr == nil {
			_ = os.Rename(backup, liveNM)
		}
		return apperr.Wrap(apperr.IO, "app.install", stageNM, err)
	}
	_ = os.RemoveAll(backup)
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
	out := make(map[string]string, len(g.Packages))
	if g == nil {
		return out
	}
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

func parsePackageSpec(spec string) (name, version string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", ""
	}
	if !strings.HasPrefix(spec, "@") {
		if i := strings.LastIndex(spec, "@"); i > 0 {
			return spec[:i], spec[i+1:]
		}
		return spec, "latest"
	}
	slash := strings.Index(spec, "/")
	if slash < 0 {
		return spec, "latest"
	}
	rest := spec[slash+1:]
	if i := strings.Index(rest, "@"); i >= 0 {
		return spec[:slash+1+i], rest[i+1:]
	}
	return spec, "latest"
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

// FormatInstallSummary returns a human-readable install summary line.
func FormatInstallSummary(r InstallResult) string {
	return fmt.Sprintf("added %d, removed %d, changed %d (%d packages)", r.Added, r.Removed, r.Changed, r.Packages)
}
