package app

import (
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/workspace"
)

func buildLocalExtractDirs(projRoot string, res *resolver.Resolution) (map[string]string, error) {
	if res == nil || !resolver.HasLocalSources(res.Extensions) {
		return nil, nil
	}
	locals, err := resolver.DecodeLocalSources(res.Extensions)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(locals))
	for key, src := range locals {
		if src.Protocol != "workspace" {
			continue
		}
		abs := filepath.Join(projRoot, filepath.FromSlash(src.Path))
		out[key] = abs
	}
	return out, nil
}

func requireWorkspacesGate(ac *Context, opts InstallOptions) error {
	if !opts.Recursive && len(opts.Filter) == 0 {
		return nil
	}
	if ac == nil || ac.Config == nil {
		return apperr.New(apperr.Internal, "app.install", "", "missing app context")
	}
	if workspace.Enabled(ac.Config) {
		return nil
	}
	return apperr.New(apperr.Usage, "app.install", "",
		"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
}

func importerIDsInGraph(g *graph.Graph) map[graph.ImporterID]bool {
	out := map[graph.ImporterID]bool{}
	if g == nil {
		return out
	}
	for _, im := range g.Importers {
		out[im.ID] = true
	}
	return out
}
