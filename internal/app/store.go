package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/store"
)

// StoreStatus summarizes the global package store.
type StoreStatus struct {
	Path         string `json:"path"`
	PackageCount int    `json:"packageCount"`
	Bytes        int64  `json:"bytes"`
}

// StoreStatusReport returns package count and bytes for m store status.
func StoreStatusReport(ctx context.Context, ac *Context) (StoreStatus, error) {
	var res StoreStatus
	if err := ctx.Err(); err != nil {
		return res, err
	}
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.store", "", "missing app context")
	}
	root, err := config.StoreRoot(ac.Config)
	if err != nil {
		return res, err
	}
	res.Path = root
	ps := store.NewPackageStore(root)
	count, bytes, err := ps.Status()
	if err != nil {
		return res, err
	}
	res.PackageCount = count
	res.Bytes = bytes
	return res, nil
}

// PruneResult reports store prune actions.
type PruneResult struct {
	Removed int      `json:"removed"`
	Kept    int      `json:"kept"`
	Paths   []string `json:"paths,omitempty"`
	DryRun  bool     `json:"dryRun"`
}

// PruneStore removes unreferenced packages from the global store.
func PruneStore(ctx context.Context, ac *Context, dryRun bool, scanRoots []string) (PruneResult, error) {
	var res PruneResult
	res.DryRun = dryRun
	if err := ctx.Err(); err != nil {
		return res, err
	}
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.store.prune", "", "missing app context")
	}
	root, err := config.StoreRoot(ac.Config)
	if err != nil {
		return res, err
	}
	if len(scanRoots) == 0 {
		scanRoots = DefaultStoreScanRoots(ac.Config.Env, "")
	}
	refs, err := CollectReferencedIntegrities(scanRoots)
	if err != nil {
		return res, err
	}
	ps := store.NewPackageStore(root)
	candidates, err := store.PruneCandidates(ps, refs)
	if err != nil {
		return res, err
	}
	for _, key := range candidates {
		path := ps.PackagePath(key)
		res.Paths = append(res.Paths, path)
		if !dryRun {
			if err := os.RemoveAll(path); err != nil {
				return res, apperr.Wrap(apperr.Store, "app.store.prune", path, err)
			}
			res.Removed++
		}
	}
	res.Kept = 0
	if keys, listErr := ps.ListPackageKeys(); listErr == nil {
		res.Kept = len(keys) - len(candidates)
	}
	if dryRun {
		res.Removed = len(res.Paths)
	}
	return res, nil
}

// FormatStoreStatus returns a human-readable status line block.
func FormatStoreStatus(s StoreStatus) string {
	return fmt.Sprintf("path=%s\npackages=%d\nbytes=%d\n", s.Path, s.PackageCount, s.Bytes)
}

// IntegrityFromKey converts algo/hex store path key to SRI integrity.
func IntegrityFromKey(algo, hex string) string {
	return algo + "-" + hex
}

// DefaultStoreScanRoots returns directories to scan for store manifests.
// Order: project root (when set), then MEW_HOME from the invocation snapshot.
func DefaultStoreScanRoots(snap config.EnvSnapshot, projectRoot string) []string {
	seen := map[string]struct{}{}
	var roots []string
	add := func(p string) {
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		roots = append(roots, abs)
	}
	if projectRoot != "" {
		add(projectRoot)
	}
	if snap.Initialized() {
		if home, ok := snap.Lookup("MEW_HOME"); ok {
			add(home)
		}
	}
	return roots
}
