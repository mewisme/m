package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
)

func newUpdateCmd() *cobra.Command {
	var (
		latest        bool
		dryRun        bool
		keepJournal   bool
		linkerMode    string
		ignoreScripts bool
		asJSON        bool
	)
	cmd := &cobra.Command{
		Use:   "update [pkg...]",
		Short: "Update dependencies and refresh the lockfile",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "update", "", "missing app context")
			}
			install := installOptsFromGlobals(cmd, app.InstallOptions{
				DryRun:        dryRun,
				KeepJournal:   keepJournal,
				Linker:        linkerMode,
				IgnoreScripts: ignoreScripts,
			})
			if len(install.Filter) > 0 {
				return apperr.New(apperr.Usage, "update", "--filter", "filtered update is not supported yet")
			}
			result, err := app.Update(cmd.Context(), ac, app.UpdateOptions{
				Targets: args,
				Latest:  latest,
				Install: install,
			})
			if err != nil {
				return err
			}
			return writeInstallResult(cmd, result, asJSON, dryRun)
		},
	}
	cmd.Flags().BoolVar(&latest, "latest", false, "bump manifest ranges to latest before resolving")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and print plan without mutating disk")
	cmd.Flags().BoolVar(&keepJournal, "journal", false, "keep transaction journal after success")
	cmd.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	cmd.Flags().BoolVar(&ignoreScripts, "ignore-scripts", false, "skip lifecycle scripts")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON (includes plan on dry-run)")
	return cmd
}
