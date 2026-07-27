package app

import (
	"context"
	"fmt"
	"os"

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
		if home := os.Getenv("MEW_HOME"); home != "" {
			scanRoots = []string{home}
		}
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
func DefaultStoreScanRoots(projRoot string) []string {
	var roots []string
	if projRoot != "" {
		roots = append(roots, projRoot)
	}
	if home := os.Getenv("MEW_HOME"); home != "" {
		roots = append(roots, home)
	}
	return roots
}
