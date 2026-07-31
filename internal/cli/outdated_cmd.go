package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/workspace"
)

func newOutdatedCmd() *cobra.Command {
	var (
		recursive bool
		asJSON    bool
	)
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Check for outdated package versions",
		Long:  "Read-only report of installed versions vs manifest ranges and registry latest. Use --recursive with workspaces to scan all importers.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "outdated", "", "missing app context")
			}
			proj, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			opts := app.OutdatedOptions{
				Recursive: recursive,
				Filter:    workspaceFilters(cmd),
			}
			if len(opts.Filter) > 0 && !workspace.Enabled(ac.Config) {
				return apperr.New(apperr.Usage, "outdated", "",
					"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
			}
			if recursive && !workspace.Enabled(ac.Config) {
				return apperr.New(apperr.Usage, "outdated", "",
					"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
			}
			report, err := app.Outdated(cmd.Context(), ac, proj, opts)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(report.Entries)
			}
			if len(report.Entries) == 0 {
				return nil
			}
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			return writeStaticOut(cmd, r.Table(outdatedTableModel(report.Entries)))
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "include all workspace importers")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print outdated report as JSON")
	return cmd
}
