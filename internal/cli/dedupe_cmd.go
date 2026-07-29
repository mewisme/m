package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newDedupeCmd() *cobra.Command {
	var (
		dryRun      bool
		prod        bool
		linkerMode  string
		asJSON      bool
		keepJournal bool
	)
	cmd := &cobra.Command{
		Use:   "dedupe",
		Short: "Reduce duplicate dependencies in the lockfile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "dedupe", "", "missing app context")
			}
			result, err := app.Dedupe(cmd.Context(), ac, app.DedupeOptions{
				DryRun: dryRun,
				InstallOptions: installOptsFromGlobals(cmd, app.InstallOptions{
					Prod:        prod,
					KeepJournal: keepJournal,
					Linker:      linkerMode,
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
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and print plan without mutating disk")
	cmd.Flags().BoolVar(&prod, "prod", false, "omit devDependencies when deduping")
	cmd.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	cmd.Flags().BoolVar(&keepJournal, "journal", false, "keep transaction journal after success")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	return cmd
}
