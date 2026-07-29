package app

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/provenance"
)

// VerifyProvenanceOptions configures provenance verification for one package.
type VerifyProvenanceOptions struct {
	PackageKey      string
	AttestationPath string
}

// VerifyProvenance checks a package provenance attestation against lock integrity.
func VerifyProvenance(ctx context.Context, ac *Context, opts VerifyProvenanceOptions) (provenance.VerifyResult, error) {
	if ac == nil {
		return provenance.VerifyResult{}, apperr.New(apperr.Internal, "app.provenance", "", "missing app context")
	}
	attPath := strings.TrimSpace(opts.AttestationPath)
	var want provenance.TarballDigest
	if key := strings.TrimSpace(opts.PackageKey); key != "" {
		proj, err := OpenProject(ctx, ac)
		if err != nil {
			return provenance.VerifyResult{}, err
		}
		pkg, err := packageFromGraph(ctx, ac, proj, key)
		if err != nil {
			return provenance.VerifyResult{}, err
		}
		want, err = provenance.DigestFromIntegrity(pkg.Integrity)
		if err != nil {
			return provenance.VerifyResult{}, apperr.Wrap(apperr.Integrity, "app.provenance", key, err)
		}
		if attPath == "" {
			attPath = defaultAttestationPath(proj.Root, pkg.ID)
		}
	}
	if attPath == "" {
		return provenance.VerifyResult{}, apperr.New(apperr.Usage, "app.provenance", "", "package key or --attestation required")
	}
	return provenance.Verify(ctx, attPath, want, provenance.VerifyOptions{})
}

func packageFromGraph(ctx context.Context, ac *Context, proj *project.Project, key string) (graph.Package, error) {
	g, err := LoadInstalledGraph(ctx, ac, proj)
	if err != nil {
		return graph.Package{}, err
	}
	for _, pkg := range g.Packages {
		if pkg.ID.Key() == key || pkg.ID.Name == key {
			return pkg, nil
		}
	}
	return graph.Package{}, apperr.New(apperr.NotFound, "app.provenance", key, "package not found in lock graph")
}

func defaultAttestationPath(projectRoot string, id graph.PackageID) string {
	name := id.Name + "-" + id.Version + ".attestation.json"
	return filepath.Join(projectRoot, name)
}
