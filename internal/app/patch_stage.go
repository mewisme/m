package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/resolver"
)

func stagePatchDerivatives(ctx context.Context, stageRoot, storeRoot string, ext lockfile.Extensions, extracts map[string]string) error {
	patches, err := resolver.DecodePatchSources(ext)
	if err != nil {
		return err
	}
	if len(patches) == 0 {
		return nil
	}
	derivRoot := filepath.Join(stageRoot, "patch-deriv")
	for pkgKey, patch := range patches {
		if err := ctx.Err(); err != nil {
			return err
		}
		src, ok := extracts[pkgKey]
		if !ok || src == "" {
			continue
		}
		if storeRoot == "" || !pathWithinRoot(src, storeRoot) {
			continue
		}
		digest := patchDigestLabel(patch.Hash)
		dest := filepath.Join(derivRoot, sanitizeKeyDir(pkgKey)+"_"+digest)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "app.install.patch", dest, err)
		}
		_ = os.RemoveAll(dest)
		if err := archive.CopyDirTree(src, dest); err != nil {
			return apperr.Wrap(apperr.IO, "app.install.patch", src, err)
		}
		extracts[pkgKey] = dest
	}
	return nil
}

func patchDigestLabel(hash string) string {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return "unknown"
	}
	if len(hash) > 16 {
		return hash[:16]
	}
	return hash
}

func pathWithinRoot(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func validatePatchPlan(g *graph.Graph, patches map[string]resolver.PatchSource, extracts map[string]string) error {
	if len(patches) == 0 {
		return nil
	}
	for pkgKey, patch := range patches {
		dir, ok := extracts[pkgKey]
		if !ok || strings.TrimSpace(dir) == "" {
			return patchPlanError("patch", pkgKey, patch, "missing writable extract directory")
		}
		if strings.TrimSpace(patch.Path) == "" {
			return patchPlanError("patch", pkgKey, patch, "missing patch file path")
		}
		fi, err := os.Stat(patch.Path)
		if err != nil {
			return patchPlanError("patch", pkgKey, patch, fmt.Sprintf("patch file: %v", err))
		}
		if fi.IsDir() {
			return patchPlanError("patch", pkgKey, patch, "patch path is a directory")
		}
		if !fi.Mode().IsRegular() {
			return patchPlanError("patch", pkgKey, patch, "patch path is not a regular file")
		}
		if g != nil && !patchSelectorMatchesGraph(g, pkgKey) {
			return patchPlanError("patch", pkgKey, patch, "patch package key does not match resolved graph package")
		}
	}
	if g != nil {
		for _, p := range g.Packages {
			if !strings.Contains(p.ID.Version, "patch_hash=") {
				continue
			}
			key := p.ID.Key()
			if _, ok := patches[key]; !ok {
				return apperr.New(apperr.Resolve, "app.install.patch", key,
					"patched graph package missing patch source")
			}
		}
	}
	return nil
}

func patchSelectorMatchesGraph(g *graph.Graph, pkgKey string) bool {
	for _, p := range g.Packages {
		if p.ID.Key() == pkgKey {
			return true
		}
	}
	return false
}

func patchPlanError(stage, pkgKey string, patch resolver.PatchSource, detail string) error {
	msg := fmt.Sprintf("%s: package %s patch %q", detail, pkgKey, patch.Path)
	if patch.Hash != "" {
		msg += fmt.Sprintf(" hash=%s", patch.Hash)
	}
	return apperr.New(apperr.Resolve, "app.install."+stage, pkgKey, msg)
}

func applyPatchesToExtracts(ctx context.Context, g *graph.Graph, ext lockfile.Extensions, extracts map[string]string) error {
	patches, err := resolver.DecodePatchSources(ext)
	if err != nil {
		return err
	}
	if err := validatePatchPlan(g, patches, extracts); err != nil {
		return err
	}
	for pkgKey, patch := range patches {
		if err := ctx.Err(); err != nil {
			return err
		}
		dir := extracts[pkgKey]
		if err := archive.ApplyUnifiedPatch(ctx, dir, patch.Path); err != nil {
			return err
		}
	}
	return nil
}
