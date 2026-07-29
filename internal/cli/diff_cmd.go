package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Diff project artifacts",
	}
	cmd.AddCommand(newDiffLockCmd())
	return cmd
}

func newDiffLockCmd() *cobra.Command {
	var (
		fromPath  string
		toPath    string
		asJSON    bool
		pnpmMajor int
	)
	cmd := &cobra.Command{
		Use:   "lock [other]",
		Short: "Diff lock graphs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "diff.lock", "", "missing app context")
			}
			opts := app.LockDiffOptions{FromPath: fromPath, ToPath: toPath, PnpmMajor: pnpmMajor}
			if len(args) > 0 {
				opts.OtherPath = args[0]
			}
			return runLockDiff(cmd.Context(), cmd.OutOrStdout(), ac, opts, asJSON)
		},
	}
	cmd.Flags().StringVar(&fromPath, "from", "", "left lockfile path")
	cmd.Flags().StringVar(&toPath, "to", "", "right lockfile path")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit diff as JSON")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")
	return cmd
}
