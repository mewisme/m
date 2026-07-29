package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newBenchCmd() *cobra.Command {
	var (
		cold     bool
		warm     bool
		asJSON   bool
		fixture  string
		baseline bool
		warmup   int
		samples  int
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
				Fixture:  fixture,
				Mode:     mode,
				Warmup:   warmup,
				Samples:  samples,
				Baseline: baseline,
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
	install.Flags().BoolVar(&baseline, "baseline", false, "update benchmarks/install-baseline.json for this case")
	install.Flags().IntVar(&warmup, "warmup", 0, "discarded warmup iterations before sampling (default 1)")
	install.Flags().IntVar(&samples, "samples", 0, "measured iterations for median/p95 (default 5)")

	cmd := &cobra.Command{
		Use:     "benchmark",
		Aliases: []string{"bench"},
		Short:   "Performance benchmarks",
	}
	cmd.AddCommand(install)
	return cmd
}

func formatBenchResult(r app.BenchResult) string {
	return fmt.Sprintf("case=%s mode=%s samples=%d medianMs=%d p95Ms=%d totalMs=%d",
		r.Case, r.Mode, len(r.Samples), r.MedianMs, r.P95Ms, r.TotalMs)
}
