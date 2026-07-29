package app

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
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
	verifyOpts, err := provenanceTrustFromConfig(ac)
	if err != nil {
		return provenance.VerifyResult{}, err
	}
	attPath := strings.TrimSpace(opts.AttestationPath)
	var want provenance.TarballDigest
	var binding *provenance.PackageBinding
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
		b := provenance.BindingFromNameVersion(pkg.ID.Name, pkg.ID.Version, want)
		binding = &b
		verifyOpts.Binding = binding
		if attPath == "" {
			attPath = defaultAttestationPath(proj.Root, pkg.ID)
		}
	}
	if attPath == "" {
		return provenance.VerifyResult{}, apperr.New(apperr.Usage, "app.provenance", "", "package key or --attestation required")
	}
	return provenance.Verify(ctx, attPath, want, verifyOpts)
}

func provenanceTrustFromConfig(ac *Context) (provenance.VerifyOptions, error) {
	keyB64 := config.String(ac.Config, "provenance.trusted_public_key", "")
	if keyB64 == "" {
		return provenance.VerifyOptions{}, apperr.New(apperr.Config, "app.provenance", "",
			"provenance trusted public key not configured; set provenance.trusted_public_key or MEW_PROVENANCE_TRUSTED_PUBLIC_KEY")
	}
	pub, err := provenance.ParsePublicKeyBase64(keyB64)
	if err != nil {
		return provenance.VerifyOptions{}, apperr.Wrap(apperr.Config, "app.provenance", "trusted_public_key", err)
	}
	return provenance.VerifyOptions{
		TrustPolicy:      provenance.TrustConfiguredKey,
		TrustedPublicKey: pub,
	}, nil
}

func packageFromGraph(ctx context.Context, ac *Context, proj *project.Project, key string) (graph.Package, error) {
	g, err := LoadInstalledGraph(ctx, ac, proj)
	if err != nil {
		return graph.Package{}, err
	}
	return packageFromGraphKey(g, key)
}

func packageFromGraphKey(g *graph.Graph, key string) (graph.Package, error) {
	if g == nil {
		return graph.Package{}, apperr.New(apperr.Internal, "app.provenance", key, "missing lock graph")
	}
	key = strings.TrimSpace(key)
	if strings.Contains(key, "@") {
		for _, pkg := range g.Packages {
			if pkg.ID.Key() == key {
				return pkg, nil
			}
		}
		return graph.Package{}, apperr.New(apperr.NotFound, "app.provenance", key, "package not found in lock graph")
	}
	var matches []graph.Package
	for _, pkg := range g.Packages {
		if pkg.ID.Name == key {
			matches = append(matches, pkg)
		}
	}
	switch len(matches) {
	case 0:
		return graph.Package{}, apperr.New(apperr.NotFound, "app.provenance", key, "package not found in lock graph")
	case 1:
		return matches[0], nil
	default:
		return graph.Package{}, apperr.New(apperr.LockAmbiguous, "app.provenance", key,
			"ambiguous package name; use name@version")
	}
}

func defaultAttestationPath(projectRoot string, id graph.PackageID) string {
	name := id.Name + "-" + id.Version + ".attestation.json"
	return filepath.Join(projectRoot, name)
}
