package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile/mlock"
	"github.com/mewisme/m/internal/resolver"
)

const lockFileName = "m.lock"

// LockPath returns the native lockfile path for a project root.
func LockPath(root string) string {
	return filepath.Join(root, lockFileName)
}

// WriteLock resolves (if needed) and writes m.lock at the project root.
func WriteLock(ctx context.Context, ac *Context, res *resolver.Resolution) error {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return err
	}
	if res == nil {
		eng, err := resolver.NewFromApp(ac.Config, proj, os.Environ())
		if err != nil {
			return err
		}
		res, err = eng.Resolve(ctx, proj.Root, resolver.ResolveOptions{})
		if err != nil {
			return err
		}
	}
	settings, err := mlock.SettingsFromEffective(ac.Config)
	if err != nil {
		return err
	}
	specs := map[graph.ImporterID][]mlock.Specifier{
		graph.RootImporter: mlock.SpecifiersFromManifest(proj.Normalized),
	}
	doc, err := mlock.FromResolution(res, specs, settings)
	if err != nil {
		return err
	}
	return mlock.WriteAtomic(LockPath(proj.Root), doc)
}

// ReadLockGraph reads and validates m.lock into a canonical graph.
func ReadLockGraph(ctx context.Context, ac *Context) (*graph.Graph, error) {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return nil, err
	}
	path := LockPath(proj.Root)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "lock.read", path, err)
	}
	doc, err := mlock.Decode(data)
	if err != nil {
		return nil, err
	}
	return mlock.ToGraph(doc)
}

// ValidateFrozenLock checks manifest specifiers match lock importers.
func ValidateFrozenLock(ctx context.Context, ac *Context) error {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return err
	}
	path := LockPath(proj.Root)
	data, err := os.ReadFile(path)
	if err != nil {
		return apperr.Wrap(apperr.IO, "lock.frozen", path, err)
	}
	doc, err := mlock.Decode(data)
	if err != nil {
		return err
	}
	manifest := map[graph.ImporterID][]mlock.Specifier{
		graph.RootImporter: mlock.SpecifiersFromManifest(proj.Normalized),
	}
	drift := mlock.ValidateFrozen(doc, manifest)
	if len(drift) == 0 {
		return nil
	}
	return apperr.New(apperr.Lockfile, "lock.frozen", path, fmt.Sprintf("manifest drift:\n%s", mlock.FormatDrift(drift)))
}
