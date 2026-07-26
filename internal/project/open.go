package project

import (
	"context"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/manifest"
)

// Open discovers the project root from cwd, detects identity, and loads package.json.
func Open(ctx context.Context, cwd string) (*Project, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := FindRoot(cwd)
	if err != nil {
		return nil, err
	}
	return openRoot(ctx, root, ".")
}

// OpenAt loads a package.json at root/rel (rel may be "." for the root package).
func OpenAt(ctx context.Context, root, rel string) (*Project, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "project.open", root, err)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "" {
		rel = "."
	}
	return openRoot(ctx, abs, rel)
}

func openRoot(ctx context.Context, root, rel string) (*Project, error) {
	_ = ctx
	dir := root
	if rel != "." {
		dir = filepath.Join(root, filepath.FromSlash(rel))
	}
	p, err := DetectIdentity(root)
	if err != nil {
		return nil, err
	}
	// For subpath importers, identity stays root-based; manifest is local.
	doc, err := manifest.LoadCached(dir)
	if err != nil {
		return nil, err
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	norm, err := manifest.ToNormalized(doc)
	if err != nil {
		return nil, err
	}
	p.Root = root
	p.Rel = rel
	p.Doc = doc
	p.Normalized = norm
	return p, nil
}
