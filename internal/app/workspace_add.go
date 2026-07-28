package app

import (
	"context"
	"path/filepath"
	"sort"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/workspace"
)

func prepareFilteredAdd(ctx context.Context, ac *Context, proj *project.Project, opts *InstallOptions, name, rng, field string) error {
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
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return apperr.Wrap(apperr.Cancelled, "app.add", name, err)
		}
		if id == graph.RootImporter {
			if err := proj.Doc.SetDependency(field, name, rng); err != nil {
				return err
			}
			continue
		}
		memPath := string(id)
		pkgPath := filepath.Join(proj.Root, filepath.FromSlash(memPath), "package.json")
		doc, err := manifest.Load(pkgPath)
		if err != nil {
			return apperr.Wrap(apperr.Manifest, "app.add", memPath, err)
		}
		if err := doc.SetDependency(field, name, rng); err != nil {
			return err
		}
		opts.MemberEdits[memPath] = doc
	}
	return nil
}

func untouchedImporterIDs(prior, active *graph.Graph) []graph.ImporterID {
	activeIDs := importerIDsInGraph(active)
	var out []graph.ImporterID
	if prior == nil {
		return out
	}
	for _, im := range prior.Importers {
		if !activeIDs[im.ID] {
			out = append(out, im.ID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// UntouchedImporterIDsForTest exports untouchedImporterIDs for tests.
func UntouchedImporterIDsForTest(prior, active *graph.Graph) []graph.ImporterID {
	return untouchedImporterIDs(prior, active)
}

// MergeFilteredWorkspaceResolutionForTest exports mergeFilteredWorkspaceResolution for tests.
func MergeFilteredWorkspaceResolutionForTest(
	prior *graph.Graph,
	priorExt lockfile.Extensions,
	active *resolver.Resolution,
	untouched []graph.ImporterID,
) (*resolver.Resolution, error) {
	return mergeFilteredWorkspaceResolution(prior, priorExt, active, untouched)
}
