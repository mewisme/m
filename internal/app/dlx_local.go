package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/runner/dlx"
)

func tryDLXLocal(ctx context.Context, ac *Context, opts DLXOptions, cwd string) (runner.ExecResult, bool, error) {
	var empty runner.ExecResult
	spec := opts.PackageSpecs[0]
	visible, err := localDependencyVisible(cwd, spec.Name)
	if err != nil {
		return empty, true, err
	}
	if !visible {
		return empty, false, nil
	}
	binNames, err := localDeclaredBins(cwd, spec.Name)
	if err != nil {
		return empty, true, err
	}
	command, err := dlx.InferModeABin(spec.Name, binNames)
	if err != nil {
		return empty, true, err
	}
	imp, err := SelectExecImporter(ctx, ac, ExecImporterOptions{})
	if err != nil {
		return empty, true, err
	}
	bind, err := LoadGenerationBinding(imp.ProjectRoot)
	if err != nil {
		return empty, true, err
	}
	res, err := runner.Exec(ctx, runner.ExecOptions{
		ProjectRoot:     imp.ProjectRoot,
		PackageDir:      imp.PackageDir,
		ImporterRel:     imp.Rel,
		NodeModules:     filepath.Join(imp.PackageDir, "node_modules"),
		Command:         command,
		PackageFilter:   spec.Name,
		ForwardedArgs:   opts.ForwardedArgs,
		HostEnv:         ac.Config.Env.Environ(),
		RequireVerified: false,
		AllowUnowned:    false,
		GenerationID:    bind.GenerationID,
		Fingerprint:     bind.Fingerprint,
		Stdin:           opts.Stdin,
		Stdout:          opts.Stdout,
		Stderr:          opts.Stderr,
	}, nil)
	return res, true, err
}

func localDependencyVisible(cwd, name string) (bool, error) {
	proj, err := project.Open(context.Background(), cwd)
	if err != nil {
		if apperr.CodeOf(err) == apperr.NotFound {
			return false, nil
		}
		return false, err
	}
	doc := proj.Doc
	if doc == nil {
		return false, nil
	}
	for _, m := range []map[string]string{doc.Dependencies, doc.DevDependencies, doc.OptionalDependencies} {
		if _, ok := m[name]; ok {
			return true, nil
		}
	}
	return false, nil
}

func localDeclaredBins(cwd, name string) ([]string, error) {
	proj, err := project.Open(context.Background(), cwd)
	if err != nil {
		return nil, err
	}
	pkgDir := filepath.Join(proj.Root, nodeModulesPackagePath(name))
	if _, err := os.Stat(pkgDir); err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.New(apperr.NotFound, "app.dlx.local", name, "package not installed locally")
		}
		return nil, apperr.Wrap(apperr.IO, "app.dlx.local", name, err)
	}
	doc, err := manifest.Load(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return nil, err
	}
	return declaredBinNames(doc)
}

func nodeModulesPackagePath(name string) string {
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			return filepath.Join("node_modules", parts[0], parts[1])
		}
	}
	return filepath.Join("node_modules", name)
}

func declaredBinNames(doc *manifest.Document) ([]string, error) {
	if doc == nil || len(doc.Bin) == 0 {
		return nil, nil
	}
	if err := manifest.ValidateBin(doc.Bin); err != nil {
		return nil, err
	}
	var single string
	if err := json.Unmarshal(doc.Bin, &single); err == nil {
		return []string{dlx.UnscopedName(doc.Name)}, nil
	}
	var obj map[string]string
	if err := json.Unmarshal(doc.Bin, &obj); err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "app.dlx.local", "bin", err)
	}
	return dlx.BinNames(obj), nil
}
