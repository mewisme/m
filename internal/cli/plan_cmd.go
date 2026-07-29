package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/plan"
)

func newPlanCmd() *cobra.Command {
	var (
		prod          bool
		frozen        bool
		linkerMode    string
		ignoreScripts bool
		recursive     bool
		asJSON        bool
		output        string
		pnpmMajor     int
	)
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Preview install mutations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "plan", "", "missing app context")
			}
			opts := installOptsFromGlobals(cmd, app.InstallOptions{
				Prod:          prod,
				Frozen:        frozen,
				Linker:        linkerMode,
				IgnoreScripts: ignoreScripts,
				Recursive:     recursive,
				PnpmMajor:     pnpmMajor,
			})
			result, err := app.PlanInstall(cmd.Context(), ac, opts)
			if err != nil {
				return err
			}
			return writePlanResult(cmd, result, asJSON, output)
		},
	}
	cmd.Flags().BoolVar(&prod, "prod", false, "omit devDependencies")
	cmd.Flags().BoolVar(&frozen, "frozen-lockfile", false, "fail if package.json and m.lock drift")
	cmd.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	cmd.Flags().BoolVar(&ignoreScripts, "ignore-scripts", false, "skip lifecycle scripts")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "install all workspace packages")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	cmd.Flags().StringVar(&output, "output", "", "write plan JSON to file")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")

	update := &cobra.Command{
		Use:   "update [pkg...]",
		Short: "Preview update mutations",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "plan update", "", "missing app context")
			}
			install := installOptsFromGlobals(cmd, app.InstallOptions{
				Linker:        linkerMode,
				IgnoreScripts: ignoreScripts,
			})
			if len(install.Filter) > 0 {
				return apperr.New(apperr.Usage, "plan update", "--filter", "filtered update is not supported yet")
			}
			result, err := app.PlanUpdate(cmd.Context(), ac, app.UpdateOptions{
				Targets: args,
				Install: install,
			})
			if err != nil {
				return err
			}
			return writePlanResult(cmd, result, asJSON, output)
		},
	}
	update.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	update.Flags().BoolVar(&ignoreScripts, "ignore-scripts", false, "skip lifecycle scripts")
	update.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	update.Flags().StringVar(&output, "output", "", "write plan JSON to file")
	cmd.AddCommand(update)
	return cmd
}

func writePlanResult(cmd *cobra.Command, result app.InstallResult, asJSON bool, output string) error {
	if output != "" {
		if result.Plan == nil {
			return apperr.New(apperr.Internal, "plan", output, "missing plan in result")
		}
		data, err := plan.EncodeJSON(result.Plan)
		if err != nil {
			return err
		}
		if err := os.WriteFile(output, data, 0o644); err != nil {
			return apperr.Wrap(apperr.IO, "plan", output, err)
		}
	}
	return writeInstallResult(cmd, result, asJSON, true)
}
