package app

import (
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/workspace"
)

func buildLocalExtractDirs(projRoot string, res *resolver.Resolution, g *graph.Graph) (map[string]string, error) {
	if res == nil && g == nil {
		return nil, nil
	}
	out := map[string]string{}
	if res != nil && resolver.HasLocalSources(res.Extensions) {
		locals, err := resolver.DecodeLocalSources(res.Extensions)
		if err != nil {
			return nil, err
		}
		for key, src := range locals {
			if src.Protocol != "workspace" {
				continue
			}
			abs := filepath.Join(projRoot, filepath.FromSlash(src.Path))
			out[key] = abs
		}
	}
	if g != nil {
		addLinkProtocolExtracts(projRoot, g, out)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func addLinkProtocolExtracts(projRoot string, g *graph.Graph, out map[string]string) {
	for _, p := range g.Packages {
		key := p.ID.Key()
		if strings.HasPrefix(key, "link:") {
			if _, ok := out[key]; !ok {
				rel := strings.TrimPrefix(key, "link:")
				out[key] = filepath.Join(projRoot, filepath.FromSlash(rel))
			}
		}
	}
	for _, e := range g.Edges {
		if !strings.HasPrefix(e.To, "link:") {
			continue
		}
		if _, ok := out[e.To]; ok {
			continue
		}
		rel := strings.TrimPrefix(e.To, "link:")
		out[e.To] = filepath.Join(projRoot, filepath.FromSlash(rel))
	}
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
