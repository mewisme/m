package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/resolver"
)

// PatchOptions controls m patch extract and commit.
type PatchOptions struct {
	Package   string
	EditDir   string
	Commit    bool
	PnpmMajor int
}

// PatchResult summarizes a patch command.
type PatchResult struct {
	EditDir   string `json:"editDir,omitempty"`
	Selector  string `json:"selector,omitempty"`
	PatchPath string `json:"patchPath,omitempty"`
}

type patchState struct {
	Selector    string `json:"selector"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	EditDir     string `json:"editDir"`
	OriginalDir string `json:"originalDir"`
}

const patchStateRel = ".mew/patch/state.json"

// Patch extracts a dependency for editing or commits a patch and reinstalls.
func Patch(ctx context.Context, ac *Context, opts PatchOptions) (PatchResult, error) {
	var res PatchResult
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.patch", "", "missing app context")
	}
	pkgName, verSpec := parsePackageSpec(strings.TrimSpace(opts.Package))
	if pkgName == "" {
		return res, apperr.New(apperr.Usage, "app.patch", opts.Package, "invalid package name")
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	pkg, selector, err := resolvePatchTarget(ctx, ac, proj, pkgName, verSpec, opts.PnpmMajor)
	if err != nil {
		return res, err
	}
	if opts.Commit {
		return patchCommit(ctx, ac, proj, opts, pkg, selector)
	}
	return patchExtract(ctx, ac, proj, opts, pkg, selector)
}

func patchExtract(ctx context.Context, ac *Context, proj *project.Project, opts PatchOptions, pkg graph.Package, selector string) (PatchResult, error) {
	var res PatchResult
	originalDir, err := fetchPatchTree(ctx, ac, proj, pkg)
	if err != nil {
		return res, err
	}
	editDir := strings.TrimSpace(opts.EditDir)
	if editDir == "" {
		editDir = defaultPatchEditDir(proj.Root, selector)
	} else if !filepath.IsAbs(editDir) {
		editDir = filepath.Join(proj.Root, filepath.FromSlash(editDir))
	}
	editAbs, err := filepath.Abs(editDir)
	if err != nil {
		return res, apperr.Wrap(apperr.IO, "app.patch", editDir, err)
	}
	if within, err := config.IsPathWithin(proj.Root, editAbs); err != nil || !within {
		return res, apperr.New(apperr.Usage, "app.patch", editDir, "edit directory must be inside project root")
	}
	_ = os.RemoveAll(editAbs)
	if err := archive.CopyDirTree(originalDir, editAbs); err != nil {
		return res, apperr.Wrap(apperr.IO, "app.patch", editAbs, err)
	}
	stateDir := filepath.Join(proj.Root, ".mew", "patch")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return res, apperr.Wrap(apperr.IO, "app.patch", stateDir, err)
	}
	state := patchState{
		Selector:    selector,
		Name:        pkg.ID.Name,
		Version:     registryBaseVersion(pkg.ID.Version),
		EditDir:     relProjectPath(proj.Root, editAbs),
		OriginalDir: relProjectPath(proj.Root, originalDir),
	}
	if err := writePatchState(proj.Root, state); err != nil {
		return res, err
	}
	res.EditDir = editAbs
	res.Selector = selector
	return res, nil
}

func patchCommit(ctx context.Context, ac *Context, proj *project.Project, opts PatchOptions, pkg graph.Package, selector string) (PatchResult, error) {
	var res PatchResult
	state, err := readPatchState(proj.Root)
	if err != nil {
		return res, err
	}
	editDir := strings.TrimSpace(opts.EditDir)
	if editDir == "" {
		if state != nil && state.Selector == selector {
			editDir = state.EditDir
		}
	}
	if editDir == "" {
		editDir = relProjectPath(proj.Root, defaultPatchEditDir(proj.Root, selector))
	}
	if !filepath.IsAbs(editDir) {
		editDir = filepath.Join(proj.Root, filepath.FromSlash(editDir))
	}
	editAbs, err := filepath.Abs(editDir)
	if err != nil {
		return res, apperr.Wrap(apperr.IO, "app.patch", editDir, err)
	}
	if _, err := os.Stat(editAbs); err != nil {
		return res, apperr.Wrap(apperr.Usage, "app.patch", editAbs, err)
	}
	originalDir := ""
	if state != nil && state.Selector == selector && state.OriginalDir != "" {
		originalDir = filepath.Join(proj.Root, filepath.FromSlash(state.OriginalDir))
	}
	if originalDir == "" || !dirExists(originalDir) {
		originalDir, err = fetchPatchTree(ctx, ac, proj, pkg)
		if err != nil {
			return res, err
		}
	}
	patchRel := filepath.ToSlash(filepath.Join("patches", selector+".patch"))
	patchAbs := filepath.Join(proj.Root, filepath.FromSlash(patchRel))
	if err := os.MkdirAll(filepath.Dir(patchAbs), 0o755); err != nil {
		return res, apperr.Wrap(apperr.IO, "app.patch", patchAbs, err)
	}
	diff, err := archive.WriteTreePatch(ctx, originalDir, editAbs)
	if err != nil {
		return res, err
	}
	if err := os.WriteFile(patchAbs, diff, 0o644); err != nil {
		return res, apperr.Wrap(apperr.IO, "app.patch", patchAbs, err)
	}
	if _, err := archive.PreflightPlan(ctx, patchAbs, originalDir); err != nil {
		_ = os.Remove(patchAbs)
		return res, err
	}
	_, err = runInstallTxn(ctx, ac, InstallOptions{
		WriteManifest: true,
		PnpmMajor:     opts.PnpmMajor,
	}, func(p *project.Project) error {
		return p.Doc.SetPatchedDependency(selector, patchRel)
	}, nil)
	if err != nil {
		return res, err
	}
	res.PatchPath = patchAbs
	res.Selector = selector
	res.EditDir = editAbs
	return res, nil
}

func resolvePatchTarget(ctx context.Context, ac *Context, proj *project.Project, pkgName, verSpec string, pnpmMajor int) (graph.Package, string, error) {
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return graph.Package{}, "", err
	}
	ropts := resolver.ResolveOptions{PnpmMajor: pnpmMajor}
	if prior, err := readLockHints(ctx, ac, proj); err == nil && prior != nil {
		ropts.Hints = prior
		ropts.Prior = prior
	}
	resolution, err := eng.Resolve(ctx, proj.Root, ropts)
	if err != nil {
		return graph.Package{}, "", err
	}
	pkg, err := findPatchPackage(resolution.Graph, pkgName, verSpec)
	if err != nil {
		return graph.Package{}, "", err
	}
	selector := pkg.ID.Name + "@" + registryBaseVersion(pkg.ID.Version)
	return pkg, selector, nil
}

func findPatchPackage(g *graph.Graph, pkgName, verSpec string) (graph.Package, error) {
	if g == nil {
		return graph.Package{}, apperr.New(apperr.NotFound, "app.patch", pkgName, "empty graph")
	}
	targetKey := ""
	for _, e := range g.Edges {
		if e.From != "." && e.From != string(graph.RootImporter) {
			continue
		}
		if e.Name != pkgName {
			continue
		}
		targetKey = e.To
		break
	}
	if targetKey == "" {
		return graph.Package{}, apperr.New(apperr.NotFound, "app.patch", pkgName, "package is not a root dependency")
	}
	for _, p := range g.Packages {
		if p.ID.Key() != targetKey {
			continue
		}
		if verSpec != "" && verSpec != "latest" {
			base := registryBaseVersion(p.ID.Version)
			if base != verSpec && !strings.HasPrefix(p.ID.Version, verSpec) {
				return graph.Package{}, apperr.New(apperr.NotFound, "app.patch", pkgName,
					fmt.Sprintf("resolved %s does not match requested %s", base, verSpec))
			}
		}
		return p, nil
	}
	return graph.Package{}, apperr.New(apperr.NotFound, "app.patch", pkgName, "resolved package missing from graph")
}

func fetchPatchTree(ctx context.Context, ac *Context, proj *project.Project, pkg graph.Package) (string, error) {
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Packages:      []graph.Package{pkg},
	}
	if err := enrichRegistryTarballs(ctx, ac, proj, g, nil); err != nil {
		return "", err
	}
	pkg = g.Packages[0]
	if strings.TrimSpace(pkg.TarballURL) == "" {
		return "", apperr.New(apperr.Resolve, "app.patch", pkg.ID.Key(), "package has no registry tarball")
	}
	stage := filepath.Join(proj.Root, ".mew", "patch", ".fetch")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return "", apperr.Wrap(apperr.IO, "app.patch", stage, err)
	}
	key := pkg.ID.Key()
	dest := filepath.Join(stage, sanitizeKeyDir(key))
	_ = os.RemoveAll(dest)
	extracts, err := fetchGraphLegacy(ctx, ac, g, stage, nil)
	if err != nil {
		return "", err
	}
	dir, ok := extracts[key]
	if !ok || dir == "" {
		return "", apperr.New(apperr.Internal, "app.patch", key, "fetch did not produce extract directory")
	}
	return dir, nil
}

func defaultPatchEditDir(projectRoot, selector string) string {
	return filepath.Join(projectRoot, ".mew", "patch", sanitizeKeyDir(selector))
}

func writePatchState(projectRoot string, state patchState) error {
	path := filepath.Join(projectRoot, filepath.FromSlash(patchStateRel))
	raw, err := json.Marshal(state)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "app.patch", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "app.patch", path, err)
	}
	return os.WriteFile(path, raw, 0o644)
}

func readPatchState(projectRoot string) (*patchState, error) {
	path := filepath.Join(projectRoot, filepath.FromSlash(patchStateRel))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "app.patch", path, err)
	}
	var state patchState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, apperr.Wrap(apperr.Usage, "app.patch", path, err)
	}
	return &state, nil
}

func relProjectPath(projectRoot, abs string) string {
	rel, err := filepath.Rel(projectRoot, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(rel)
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
