package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

// InstallOptions controls m install / ci.
type InstallOptions struct {
	Prod        bool
	Frozen      bool
	DryRun      bool
	KeepJournal bool
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

// Install resolves, fetches, links, and commits via journaled transaction.
func Install(ctx context.Context, ac *Context, opts InstallOptions) (InstallResult, error) {
	return runInstallTxn(ctx, ac, opts, nil)
}

// Add declares a dependency in memory and installs (manifest written at commit).
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
	edit := func(p *project.Project) error {
		return p.Doc.SetDependency(field, name, rng)
	}
	return runInstallTxn(ctx, ac, opts.Install, edit)
}

// Remove deletes a dependency in memory and reinstalls (manifest written at commit).
func Remove(ctx context.Context, ac *Context, name string, opts InstallOptions) (InstallResult, error) {
	fields := []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}
	edit := func(p *project.Project) error {
		var removed bool
		for _, field := range fields {
			if err := p.Doc.RemoveDependency(field, name); err != nil {
				if apperr.CodeOf(err) == apperr.NotFound {
					continue
				}
				return err
			}
			removed = true
			break
		}
		if !removed {
			return apperr.New(apperr.NotFound, "app.remove", name, "dependency not found")
		}
		return nil
	}
	return runInstallTxn(ctx, ac, opts, edit)
}

// FormatInstallSummary returns a human-readable install summary line.
func FormatInstallSummary(r InstallResult) string {
	return fmt.Sprintf("added %d, removed %d, changed %d (%d packages)", r.Added, r.Removed, r.Changed, r.Packages)
}

// ponytail: fetch/link helpers remain here; commit path is install_txn.go
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
