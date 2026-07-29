package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newPruneCmd() *cobra.Command {
	var (
		prod        bool
		dryRun      bool
		linkerMode  string
		asJSON      bool
		keepJournal bool
	)
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove extraneous packages from node_modules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "prune", "", "missing app context")
			}
			result, err := app.PruneNodeModules(cmd.Context(), ac, app.PruneOptions{
				Prod:   prod,
				DryRun: dryRun,
				InstallOptions: installOptsFromGlobals(cmd, app.InstallOptions{
					Linker:      linkerMode,
					KeepJournal: keepJournal,
				}),
			})
			outErr := writeInstallResult(cmd, result, asJSON, dryRun)
			if err != nil {
				if outErr != nil {
					return outErr
				}
				return err
			}
			return outErr
		},
	}
	cmd.Flags().BoolVar(&prod, "prod", false, "prune dev-only extraneous packages")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list removals without mutating disk")
	cmd.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	cmd.Flags().BoolVar(&keepJournal, "journal", false, "keep transaction journal after success")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	return cmd
}
