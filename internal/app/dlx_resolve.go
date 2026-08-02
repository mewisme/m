package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/runner/dlx"
	"github.com/mewisme/mew/internal/snapshot"
)

type dlxResolveResult struct {
	Identity     dlx.ResolvedEnvironmentIdentity
	DirectKeys   []string
	DirectBins   map[string]map[string]string
	Resolution   *resolver.Resolution
	EphemeralDir string
	TxnID        string
}

func buildRequestIdentity(ac *Context, opts DLXOptions) dlx.RequestIdentity {
	return dlx.RequestIdentity{
		NormalizedSpecs:            dlx.SortSpecs(opts.PackageSpecs),
		SanitizedRegistryOrigin:    dlx.SanitizeRegistryOrigin(config.String(ac.Config, "registry", "")),
		ResolverPolicyFingerprint:  resolverPolicyFingerprint(ac),
		TargetPlatformFingerprint:  runtime.GOOS + "/" + runtime.GOARCH,
		LinkerMode:                 config.String(ac.Config, "install.linker", "auto"),
		LifecyclePolicyFingerprint: lifecyclePolicyFingerprint(ac),
	}
}

func resolverPolicyFingerprint(ac *Context) string {
	if ac == nil || ac.Config == nil {
		return ""
	}
	if config.Bool(ac.Config, "resolve.strict_peer_dependencies", true) {
		return "strict-peers"
	}
	return "loose-peers"
}

func lifecyclePolicyFingerprint(ac *Context) string {
	if ac == nil || ac.Config == nil {
		return "deny"
	}
	return config.String(ac.Config, "lifecycle.script_trust", "deny")
}

func resolveDLXMetadata(ctx context.Context, ac *Context, opts DLXOptions) (*dlxResolveResult, dlx.RequestIdentity, error) {
	reqID := buildRequestIdentity(ac, opts)
	ephemeral, err := prepareEphemeralProject(opts)
	if err != nil {
		return nil, reqID, err
	}
	eng, err := resolver.NewFromApp(ac.Config, ephemeral)
	if err != nil {
		return nil, reqID, err
	}
	res, err := eng.ResolveProject(ctx, ephemeral, resolver.ResolveOptions{
		Policy: resolver.PolicyFromEffective(ac.Config),
	})
	if err != nil {
		return nil, reqID, err
	}
	graphDigest, err := snapshot.GraphDigest(res.Graph)
	if err != nil {
		return nil, reqID, err
	}
	pkgs := make([]dlx.ResolvedPackage, 0, len(res.Graph.Packages))
	for _, p := range res.Graph.Packages {
		pkgs = append(pkgs, dlx.ResolvedPackage{
			Name: p.ID.Name, Version: p.ID.Version, Integrity: p.Integrity, Key: p.ID.Key(),
		})
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Key < pkgs[j].Key })
	identity := dlx.ResolvedEnvironmentIdentity{
		GraphDigest:                graphDigest,
		Packages:                   pkgs,
		SanitizedRegistryOrigin:    reqID.SanitizedRegistryOrigin,
		TargetPlatformFingerprint:  reqID.TargetPlatformFingerprint,
		NodeFingerprint:            reqID.NodeFingerprint,
		LinkerMode:                 reqID.LinkerMode,
		LifecyclePolicyFingerprint: reqID.LifecyclePolicyFingerprint,
		ResolverPolicyFingerprint:  reqID.ResolverPolicyFingerprint,
	}
	direct := directPackageNames(opts.PackageSpecs)
	directBins, err := binsFromResolution(ctx, ac, ephemeral, res, direct)
	if err != nil {
		return nil, reqID, err
	}
	txnID, err := fsx.NewLockID()
	if err != nil {
		return nil, reqID, err
	}
	out := &dlxResolveResult{
		Identity:     identity,
		DirectKeys:   direct,
		DirectBins:   directBins,
		Resolution:   res,
		EphemeralDir: ephemeral.Root,
		TxnID:        txnID,
	}
	return out, reqID, nil
}

func prepareEphemeralProject(opts DLXOptions) (*project.Project, error) {
	root, err := os.MkdirTemp("", "mew-mx-resolve-*")
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "app.dlx.resolve", "", err)
	}
	deps := map[string]string{}
	for _, s := range opts.PackageSpecs {
		ver := s.Version
		if ver == "" {
			ver = "latest"
		}
		deps[s.Name] = ver
	}
	raw, err := json.Marshal(map[string]any{
		"name":         "mx-ephemeral",
		"version":      "0.0.0",
		"private":      true,
		"dependencies": deps,
	})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), raw, 0o644); err != nil {
		return nil, apperr.Wrap(apperr.IO, "app.dlx.resolve", root, err)
	}
	return project.Open(context.Background(), root)
}

func directPackageNames(specs []dlx.PackageSpec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	sort.Strings(out)
	return out
}

func binsFromResolution(ctx context.Context, ac *Context, proj *project.Project, res *resolver.Resolution, direct []string) (map[string]map[string]string, error) {
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return nil, err
	}
	owners := map[string]map[string]string{}
	for _, name := range direct {
		base := registry.ResolveBaseForPackage(ac.Config, proj.Root, proj.Identity, name)
		pack, err := eng.Client.Packument(ctx, base, name)
		if err != nil {
			return nil, err
		}
		ver := ""
		for _, p := range res.Graph.Packages {
			if p.ID.Name == name {
				ver = p.ID.Version
				break
			}
		}
		meta, err := pack.SelectVersion(ver)
		if err != nil {
			return nil, err
		}
		bins := map[string]string{}
		if len(meta.Bin) == 1 {
			for k, v := range meta.Bin {
				bins[k] = v
			}
		} else if len(meta.Bin) > 1 {
			bins = meta.Bin
		} else {
			bins[dlx.UnscopedName(name)] = ""
		}
		owners[name] = bins
	}
	return owners, nil
}

func writeResolvedLock(path string, ac *Context, res *resolver.Resolution) error {
	settings, err := mlock.SettingsWithFingerprints(ac.Config, nil)
	if err != nil {
		return err
	}
	specs := map[graph.ImporterID][]mlock.Specifier{graph.ImporterID("."): {}}
	doc, err := mlock.FromResolution(res, specs, settings)
	if err != nil {
		return err
	}
	return mlock.WriteAtomic(path, doc)
}
