package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/resolver"
)

// LockPath returns the incumbent lockfile path for a project.
func LockPath(proj *project.Project) string {
	name := project.LockFilename(proj.Identity)
	if name == "" {
		return filepath.Join(proj.Root, "m.lock")
	}
	return filepath.Join(proj.Root, name)
}

// IncumbentLockBasename returns the lockfile basename for project identity.
func IncumbentLockBasename(proj *project.Project) string {
	name := project.LockFilename(proj.Identity)
	if name == "" {
		return "m.lock"
	}
	return name
}

// WriteLock resolves (if needed) and writes the incumbent lockfile at the project root.
func WriteLock(ctx context.Context, ac *Context, res *resolver.Resolution) error {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return err
	}
	if res == nil {
		eng, err := resolver.NewFromApp(ac.Config, proj)
		if err != nil {
			return err
		}
		res, err = eng.Resolve(ctx, proj.Root, resolver.ResolveOptions{})
		if err != nil {
			return err
		}
	}
	switch proj.Identity {
	case project.IdentityMew:
		settings, err := mlock.SettingsWithFingerprints(ac.Config, proj.Normalized.Overrides)
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
		return mlock.WriteAtomic(LockPath(proj), doc)
	case project.IdentityNub, project.IdentityPNPM:
		ext, ok := lockfile.ExtAdapterFor(proj.Identity)
		if !ok {
			return lockfile.NewUnsupported("lock.write", project.LockFilename(proj.Identity), "adapter not registered")
		}
		prior, err := project.ReadLockfileBytes(proj.Root, proj.Identity)
		if err != nil {
			return err
		}
		det := lockfile.Detection{}
		if proj.Identity == project.IdentityPNPM {
			det, err = lockfile.DetectPnpm(prior)
			if err != nil {
				return err
			}
		}
		out, err := lockfile.EncodePreserving(ctx, ext, LockPath(proj), res.Graph, prior, nil, det)
		if err != nil {
			return err
		}
		if out.Unchanged {
			return nil
		}
		return os.WriteFile(LockPath(proj), out.Bytes, 0o644)
	default:
		return lockfile.NewUnsupported("lock.write", string(proj.Identity), "lock adapter not implemented")
	}
}

// ReadLockSettings reads m.lock settings without full graph conversion.
func ReadLockSettings(root string, id project.Identity) (mlock.Settings, error) {
	doc, err := readLockDocument(root, id)
	if err != nil {
		return mlock.Settings{}, err
	}
	return doc.Settings, nil
}

func readLockSettings(root string, id project.Identity) (*mlock.Document, error) {
	return readLockDocument(root, id)
}

func readLockDocument(root string, id project.Identity) (*mlock.Document, error) {
	if id != project.IdentityMew {
		return nil, nil
	}
	path := filepath.Join(root, "m.lock")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "lock.read", path, err)
	}
	return mlock.Decode(data)
}

// ReadLockGraph reads and validates the incumbent lock into a canonical graph.
func ReadLockGraph(ctx context.Context, ac *Context) (*graph.Graph, error) {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return nil, err
	}
	return lockfile.ReadGraph(ctx, proj.Root, proj.Identity)
}

// ValidateFrozenLock checks manifest specifiers match lock importers.
func ValidateFrozenLock(ctx context.Context, ac *Context) error {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return err
	}
	return validateFrozenLockForProject(ctx, ac, proj)
}

func validateFrozenLockForProject(ctx context.Context, ac *Context, proj *project.Project) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_ = ac
	switch proj.Identity {
	case project.IdentityMew:
		path := LockPath(proj)
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
	default:
		g, err := lockfile.ReadGraph(ctx, proj.Root, proj.Identity)
		if err != nil {
			return err
		}
		return validateFrozenFromGraph(proj, g)
	}
}

func validateFrozenFromGraph(proj *project.Project, g *graph.Graph) error {
	manifest := map[graph.ImporterID][]mlock.Specifier{
		graph.RootImporter: mlock.SpecifiersFromManifest(proj.Normalized),
	}
	lockSections := []mlock.ImporterSection{{
		ID:         graph.RootImporter,
		Specifiers: specifiersFromGraph(g, graph.RootImporter),
	}}
	drift := mlock.CompareSpecifiers(lockSections, manifest)
	if len(drift) == 0 {
		return nil
	}
	return apperr.New(apperr.Lockfile, "lock.frozen", LockPath(proj), fmt.Sprintf("manifest drift:\n%s", mlock.FormatDrift(drift)))
}

func specifiersFromGraph(g *graph.Graph, importer graph.ImporterID) []mlock.Specifier {
	if g == nil {
		return nil
	}
	var out []mlock.Specifier
	for _, e := range g.Edges {
		if e.From != string(importer) || e.Range == "" {
			continue
		}
		out = append(out, mlock.Specifier{Name: e.Name, Range: e.Range, Kind: e.Kind})
	}
	return out
}
