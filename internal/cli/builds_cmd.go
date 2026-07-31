package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lifecycle"
)

func newBuildsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "builds",
		Short: "Lifecycle build script audit",
	}
	cmd.AddCommand(newBuildsListCmd())
	return cmd
}

func newBuildsListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List lifecycle script executions from the audit log",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "builds.list", "", "missing app context")
			}
			entries, err := lifecycle.ReadAudit(lifecycle.AuditFilePath(ac.CWD))
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}
			if len(entries) == 0 {
				g := ownerFlags(cmd.Root())
				r := g.mustStaticRenderer(cmd)
				return writeStaticOut(cmd, r.Notice(emptyNotice("no lifecycle audit entries")))
			}
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			return writeStaticOut(cmd, r.Table(buildsTableModel(entries)))
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print audit entries as JSON")
	return cmd
}
