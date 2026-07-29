package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/snapshot"
)

func newHistoryCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show install snapshot timeline",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "history", "", "missing app context")
			}
			proj, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			list, err := snapshot.NewStore(proj.Root).List()
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(snapshotEntriesWithDelta(list))
			}
			if len(list) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "no snapshots")
				return err
			}
			for i, s := range list {
				var older *snapshot.Snapshot
				if i+1 < len(list) {
					older = &list[i+1]
				}
				if err := formatSnapshotLine(cmd.OutOrStdout(), s, older); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print timeline as JSON")
	return cmd
}
