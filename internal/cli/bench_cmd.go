package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newBenchCmd() *cobra.Command {
	var (
		cold    bool
		warm    bool
		asJSON  bool
		fixture string
	)
	install := &cobra.Command{
		Use:   "install",
		Short: "Benchmark install against a fixture project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "bench install", "", "missing app context")
			}
			mode := app.BenchCold
			if warm {
				mode = app.BenchWarm
			}
			if cold && warm {
				return apperr.New(apperr.Usage, "bench install", "--cold|--warm", "specify only one mode flag")
			}
			result, err := app.BenchInstall(cmd.Context(), ac, app.BenchInstallOptions{
				Fixture: fixture,
				Mode:    mode,
			})
			if err != nil {
				return err
			}
			if asJSON {
				data, err := app.EncodeBenchResultJSON(result)
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(data)
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write([]byte("\n"))
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(formatBenchResult(result) + "\n"))
			return err
		},
	}
	install.Flags().BoolVar(&cold, "cold", false, "clear isolated cache and store before install")
	install.Flags().BoolVar(&warm, "warm", false, "reuse cache from prior bench run in bench home")
	install.Flags().BoolVar(&asJSON, "json", false, "emit BenchResult JSON")
	install.Flags().StringVar(&fixture, "fixture", "", "fixture project path (default fixtures/bench/medium-graph)")

	cmd := &cobra.Command{
		Use:     "benchmark",
		Aliases: []string{"bench"},
		Short:   "Performance benchmarks",
	}
	cmd.AddCommand(install)
	return cmd
}

func formatBenchResult(r app.BenchResult) string {
	return fmt.Sprintf("case=%s mode=%s totalMs=%d", r.Case, r.Mode, r.TotalMs)
}
