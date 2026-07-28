package app

import (
	"context"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/workspace"
)

func prepareFilteredRemove(ctx context.Context, ac *Context, proj *project.Project, opts *InstallOptions, name string) error {
	if err := requireWorkspacesGate(ac, *opts); err != nil {
		return err
	}
	wg, err := workspace.BuildGraph(proj.Root)
	if err != nil {
		return err
	}
	ids, err := workspace.ExpandFilter(wg, opts.Filter)
	if err != nil {
		return err
	}
	if opts.MemberEdits == nil {
		opts.MemberEdits = map[string]*manifest.Document{}
	}
	fields := []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}
	var removed bool
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return apperr.Wrap(apperr.Cancelled, "app.remove", name, err)
		}
		if id == graph.RootImporter {
			if ok, err := removeFromDoc(proj.Doc, fields, name); err != nil {
				return err
			} else if ok {
				removed = true
			}
			continue
		}
		memPath := string(id)
		doc := opts.MemberEdits[memPath]
		if doc == nil {
			pkgPath := filepath.Join(proj.Root, filepath.FromSlash(memPath), "package.json")
			doc, err = manifest.Load(pkgPath)
			if err != nil {
				return apperr.Wrap(apperr.Manifest, "app.remove", memPath, err)
			}
		}
		ok, err := removeFromDoc(doc, fields, name)
		if err != nil {
			return err
		}
		if ok {
			removed = true
			opts.MemberEdits[memPath] = doc
		}
	}
	if !removed {
		return apperr.New(apperr.NotFound, "app.remove", name, "dependency not found")
	}
	return nil
}

func removeFromDoc(doc *manifest.Document, fields []string, name string) (bool, error) {
	for _, field := range fields {
		if err := doc.RemoveDependency(field, name); err != nil {
			if apperr.CodeOf(err) == apperr.NotFound {
				continue
			}
			return false, err
		}
		return true, nil
	}
	return false, nil
}
