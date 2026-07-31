package cli

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/workspace"
)

func newLsCmd() *cobra.Command {
	var (
		recursive bool
		depth     int
		asJSON    bool
		prodOnly  bool
	)
	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List dependency tree or workspace packages",
		Long:    "Default: print the lockfile dependency tree for the root importer. With workspaces enabled, -r lists workspace member packages instead.",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "ls", "", "missing app context")
			}
			proj, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			if recursive && workspace.Enabled(ac.Config) {
				return runLsWorkspace(cmd, ac, proj, recursive, depth)
			}
			if recursive && !workspace.Enabled(ac.Config) {
				return apperr.New(apperr.Usage, "ls", "",
					"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
			}
			if len(workspaceFilters(cmd)) > 0 && !workspace.Enabled(ac.Config) {
				return apperr.New(apperr.Usage, "ls", "",
					"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
			}
			return runLsDepTree(cmd, proj, depth, prodOnly, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "list all workspace packages (requires workspaces)")
	cmd.Flags().IntVar(&depth, "depth", -1, "max depth (dependency tree levels or workspace path depth)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print dependency tree as JSON")
	cmd.Flags().BoolVar(&prodOnly, "prod", false, "omit dev dependency edges from the tree")
	return cmd
}

func runLsDepTree(cmd *cobra.Command, proj *project.Project, depth int, prodOnly, asJSON bool) error {
	ac := app.FromContext(cmd.Context())
	g, err := app.LoadInstalledGraph(cmd.Context(), ac, proj)
	if err != nil {
		return err
	}
	treeOpts := app.DepTreeOptions{
		ImporterID: graph.RootImporter,
		ProdOnly:   prodOnly,
		Depth:      depth,
	}
	tree, err := app.BuildDepTree(g, treeOpts)
	if err != nil {
		return err
	}
	rootName, rootVersion := app.ImporterLabel(proj, graph.RootImporter)
	if asJSON {
		doc := map[string]any{
			"name":         rootName,
			"version":      rootVersion,
			"dependencies": tree.Dependencies,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(doc)
	}
	text := app.FormatDepTreeText(rootName, rootVersion, tree)
	if text == "" {
		return nil
	}
	gf := ownerFlags(cmd.Root())
	r := gf.mustStaticRenderer(cmd)
	return writeStaticOut(cmd, r.PlainText(text))
}

func runLsWorkspace(cmd *cobra.Command, ac *app.Context, proj *project.Project, recursive bool, depth int) error {
	if len(workspaceFilters(cmd)) > 0 && !workspace.Enabled(ac.Config) {
		return apperr.New(apperr.Usage, "ls", "",
			"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
	}
	patterns, err := proj.Doc.WorkspacePatterns()
	if err != nil {
		return err
	}
	if len(patterns) == 0 {
		g := ownerFlags(cmd.Root())
		r := g.mustStaticRenderer(cmd)
		return writeStaticOut(cmd, r.KeyValues([]presentation.KeyValue{
			{Key: "name", Value: proj.Normalized.Name, Style: presentation.ValuePackage},
			{Key: "version", Value: proj.Doc.Version, Style: presentation.ValueVersion},
		}))
	}
	wg, err := workspace.BuildGraph(proj.Root)
	if err != nil {
		return err
	}
	var paths []string
	if filters := workspaceFilters(cmd); len(filters) > 0 {
		ids, err := workspace.ExpandFilter(wg, filters)
		if err != nil {
			return err
		}
		for _, id := range ids {
			paths = append(paths, string(id))
		}
	} else if recursive {
		paths = wg.MemberPaths()
	} else {
		g := ownerFlags(cmd.Root())
		r := g.mustStaticRenderer(cmd)
		return writeStaticOut(cmd, r.KeyValues([]presentation.KeyValue{
			{Key: "name", Value: proj.Normalized.Name, Style: presentation.ValuePackage},
			{Key: "version", Value: proj.Doc.Version, Style: presentation.ValueVersion},
			{Key: "path", Value: ".", Style: presentation.ValuePath},
		}))
	}
	var rows []workspaceListRow
	for _, p := range paths {
		if p == "." {
			continue
		}
		if depth >= 0 && pathDepth(p) > depth {
			continue
		}
		mem, ok := wg.ByPath[p]
		if !ok {
			continue
		}
		rows = append(rows, workspaceListRow{Name: mem.Name, Version: mem.Version, Path: p})
	}
	g := ownerFlags(cmd.Root())
	r := g.mustStaticRenderer(cmd)
	return writeStaticOut(cmd, r.Table(workspaceListTable(rows)))
}

func pathDepth(p string) int {
	p = strings.Trim(p, "/")
	if p == "" || p == "." {
		return 0
	}
	return strings.Count(p, "/") + 1
}
