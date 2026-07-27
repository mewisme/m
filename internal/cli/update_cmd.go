package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
)

func newUpdateCmd() *cobra.Command {
	var latest bool
	cmd := &cobra.Command{
		Use:   "update [pkg...]",
		Short: "Update dependencies and refresh the lockfile",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "update", "", "missing app context")
			}
			return app.Update(cmd.Context(), ac, app.UpdateOptions{
				Targets: args,
				Latest:  latest,
			})
		},
	}
	cmd.Flags().BoolVar(&latest, "latest", false, "bump manifest ranges to latest before resolving")
	return cmd
}
