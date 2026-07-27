package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/workspace"
)

func newLsCmd() *cobra.Command {
	var recursive bool
	var depth int
	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List workspace packages",
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
			if recursive && !workspace.Enabled(ac.Config) {
				return apperr.New(apperr.Usage, "ls", "",
					"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
			}
			if len(workspaceFilters(cmd)) > 0 && !workspace.Enabled(ac.Config) {
				return apperr.New(apperr.Usage, "ls", "",
					"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
			}
			patterns, err := proj.Doc.WorkspacePatterns()
			if err != nil {
				return err
			}
			if len(patterns) == 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", proj.Normalized.Name, proj.Doc.Version)
				return err
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
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s .\n", proj.Normalized.Name, proj.Doc.Version)
				return err
			}
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
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s\n", mem.Name, mem.Version, p)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "list all workspace packages")
	cmd.Flags().IntVar(&depth, "depth", -1, "max directory depth under workspace root")
	return cmd
}

func pathDepth(p string) int {
	p = strings.Trim(p, "/")
	if p == "" || p == "." {
		return 0
	}
	return strings.Count(p, "/") + 1
}
