package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/pack"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/registry"
)

// ProvenanceAttest is an optional publish provenance hook (no-op when nil).
type ProvenanceAttest func(ctx context.Context, meta PublishMeta) ([]byte, error)

// PublishMeta is input to the provenance hook.
type PublishMeta struct {
	Name    string
	Version string
	Tarball string
}

// PublishOptions configures m publish.
type PublishOptions struct {
	TarballPath      string
	DryRun           bool
	Tag              string
	Access           string
	OTP              string
	Provenance       bool
	ProvenanceAttest ProvenanceAttest
	PackDestination  string
	PackageDir       string
}

// PublishResult summarizes publish or dry-run.
type PublishResult struct {
	DryRun      bool   `json:"dryRun"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Registry    string `json:"registry"`
	Tag         string `json:"tag"`
	Access      string `json:"access,omitempty"`
	Tarball     string `json:"tarball"`
	TarballSize int    `json:"tarballBytes"`
	Provenance  string `json:"provenance,omitempty"`
}

// Publish packs (unless a tarball is given) and PUTs to the registry.
func Publish(ctx context.Context, ac *Context, opts PublishOptions) (PublishResult, error) {
	if ac == nil {
		return PublishResult{}, apperr.New(apperr.Internal, "app.publish", "", "missing app context")
	}
	tgzPath, err := ResolvePackTarball(ctx, ac, opts.TarballPath, PackOptions{
		PackDestination: opts.PackDestination,
		PackageDir:      opts.PackageDir,
	})
	if err != nil {
		return PublishResult{}, err
	}

	pkgJSON, err := pack.ReadPackageJSONFromTarball(tgzPath)
	if err != nil {
		return PublishResult{}, err
	}
	norm, err := pack.NormalizePackageJSON(pkgJSON)
	if err != nil {
		return PublishResult{}, err
	}
	var doc struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(norm, &doc); err != nil {
		return PublishResult{}, apperr.Wrap(apperr.Manifest, "app.publish", "package.json", err)
	}
	if doc.Name == "" || doc.Version == "" {
		return PublishResult{}, apperr.New(apperr.Manifest, "app.publish", "package.json", "name and version are required")
	}
	if err := pack.ValidateTarballName(tgzPath, doc.Name, doc.Version); err != nil {
		return PublishResult{}, err
	}

	root, err := project.FindRoot(ac.CWD)
	if err != nil {
		return PublishResult{}, err
	}
	proj, err := project.Open(ctx, ac.CWD)
	if err != nil {
		return PublishResult{}, err
	}
	base := strings.TrimRight(registry.ResolveBaseForPackage(ac.Config, root, proj.Identity, doc.Name), "/")

	tag := opts.Tag
	if tag == "" {
		tag = "latest"
	}
	access := opts.Access
	if access != "" && access != "public" && access != "restricted" {
		return PublishResult{}, apperr.New(apperr.Usage, "app.publish", "access", `must be "public" or "restricted"`)
	}

	tarball, err := pack.TarballBytes(tgzPath)
	if err != nil {
		return PublishResult{}, err
	}

	result := PublishResult{
		DryRun:      opts.DryRun,
		Name:        doc.Name,
		Version:     doc.Version,
		Registry:    base,
		Tag:         tag,
		Access:      access,
		Tarball:     tgzPath,
		TarballSize: len(tarball),
	}

	if opts.Provenance && opts.ProvenanceAttest != nil {
		att, err := opts.ProvenanceAttest(ctx, PublishMeta{
			Name: doc.Name, Version: doc.Version, Tarball: tgzPath,
		})
		if err != nil {
			return PublishResult{}, apperr.Wrap(apperr.Internal, "app.publish", "provenance", err)
		}
		if len(att) > 0 {
			result.Provenance = filepath.Join(filepath.Dir(tgzPath), doc.Name+"-"+doc.Version+".provenance.json")
		}
	}

	if opts.DryRun {
		return result, nil
	}

	timeoutMs := config.Int(ac.Config, "network.timeout_ms", 60000)
	hc, err := fetch.NewClient(fetch.Options{
		Timeout:  time.Duration(timeoutMs) * time.Millisecond,
		ProxyURL: config.String(ac.Config, "network.proxy", ""),
		CAFile:   config.String(ac.Config, "network.ca_file", ""),
	})
	if err != nil {
		return PublishResult{}, err
	}

	_, err = registry.Publish(ctx, registry.PublishOptions{
		RegistryBase: base,
		Name:         doc.Name,
		Version:      doc.Version,
		Tag:          tag,
		Access:       access,
		OTP:          opts.OTP,
		AuthToken:    config.AuthToken(ac.Config),
		Tarball:      tarball,
		PackageJSON:  norm,
		HTTPClient:   hc,
	})
	if err != nil {
		return PublishResult{}, err
	}
	return result, nil
}

// FormatPublishPlan returns a human-readable dry-run line.
func FormatPublishPlan(r PublishResult) string {
	return fmt.Sprintf("publish %s@%s → %s tag=%s (%d bytes)%s\n",
		r.Name, r.Version, r.Registry, r.Tag, r.TarballSize, dryRunSuffix(r.DryRun))
}

func dryRunSuffix(dry bool) string {
	if dry {
		return " (dry-run)"
	}
	return ""
}

// FormatPublishSuccess returns stdout after a successful publish.
func FormatPublishSuccess(r PublishResult) string {
	return fmt.Sprintf("+ %s@%s\n", r.Name, r.Version)
}
