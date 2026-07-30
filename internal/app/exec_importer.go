package app

import (
	"context"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/workspace"
)

// ExecImporter is one workspace importer selected for m exec.
type ExecImporter struct {
	ProjectRoot string
	Rel         string
	PackageDir  string
}

// ExecImporterOptions configures single-importer selection.
type ExecImporterOptions struct {
	Recursive bool
	Filters   []string
}

// SelectExecImporter resolves exactly one importer for bin execution.
func SelectExecImporter(ctx context.Context, ac *Context, opts ExecImporterOptions) (ExecImporter, error) {
	if ac == nil {
		return ExecImporter{}, apperr.New(apperr.Internal, "app.exec.importer", "", "missing app context")
	}
	if opts.Recursive {
		return ExecImporter{}, apperr.New(apperr.Usage, "app.exec.importer", "", "m exec does not support -r/--recursive")
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return ExecImporter{}, err
	}
	packageDir := proj.Root
	rel := "."
	if proj.Rel != "." && proj.Rel != "" {
		rel = filepath.ToSlash(proj.Rel)
		packageDir = filepath.Join(proj.Root, filepath.FromSlash(proj.Rel))
	}
	if len(opts.Filters) > 0 {
		if !workspace.Enabled(ac.Config) {
			return ExecImporter{}, apperr.New(apperr.Usage, "app.exec.importer", "", "workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
		}
		wg, err := workspace.BuildGraph(proj.Root)
		if err != nil {
			return ExecImporter{}, err
		}
		paths, err := runner.SelectMembers(wg, false, opts.Filters)
		if err != nil {
			return ExecImporter{}, err
		}
		if len(paths) != 1 {
			if len(paths) == 0 {
				return ExecImporter{}, apperr.New(apperr.NotFound, "app.exec.importer", "", "no workspace packages matched --filter")
			}
			return ExecImporter{}, apperr.New(apperr.Usage, "app.exec.importer", "", "m exec --filter requires exactly one workspace member")
		}
		rel = paths[0]
		packageDir = filepath.Join(proj.Root, filepath.FromSlash(rel))
	}
	return ExecImporter{ProjectRoot: proj.Root, Rel: rel, PackageDir: packageDir}, nil
}
