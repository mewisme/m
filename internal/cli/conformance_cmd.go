package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/conformance"
)

func newConformanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conformance",
		Short: "Run certification conformance suites",
		Long:  "List and execute the PM core certification matrix.",
	}
	cmd.AddCommand(newConformanceListCmd())
	cmd.AddCommand(newConformanceRunCmd())
	return cmd
}

func newConformanceListCmd() *cobra.Command {
	var filter string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List core certification suites",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := findModuleRoot()
			if err != nil {
				return apperr.Wrap(apperr.Internal, "conformance list", "", err)
			}
			suites, err := conformance.ListCore(repoRoot, filter)
			if err != nil {
				return apperr.Wrap(apperr.Usage, "conformance list", "", err)
			}
			for _, s := range suites {
				req := "optional"
				if s.Required {
					req = "required"
				}
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", s.ID, req, s.Package, s.Run)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&filter, "filter", "", "suite id prefix filter")
	return cmd
}

func newConformanceRunCmd() *cobra.Command {
	var (
		asJSON bool
		filter string
	)
	run := &cobra.Command{
		Use:   "core",
		Short: "Run the PM core certification matrix",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := findModuleRoot()
			if err != nil {
				return apperr.Wrap(apperr.Internal, "conformance run core", "", err)
			}
			report, err := conformance.RunCore(cmd.Context(), conformance.RunOptions{
				RepoRoot: repoRoot,
				Filter:   filter,
			})
			if asJSON {
				data, encErr := report.EncodeJSON()
				if encErr != nil {
					return apperr.Wrap(apperr.Internal, "conformance run core", "", encErr)
				}
				if _, writeErr := cmd.OutOrStdout().Write(data); writeErr != nil {
					return writeErr
				}
				_, writeErr := cmd.OutOrStdout().Write([]byte("\n"))
				if writeErr != nil {
					return writeErr
				}
			} else {
				for _, s := range report.Suites {
					line := fmt.Sprintf("%s\t%s", s.ID, s.Status)
					if s.Error != "" {
						line += "\t" + s.Error
					}
					if _, writeErr := fmt.Fprintln(cmd.OutOrStdout(), line); writeErr != nil {
						return writeErr
					}
				}
				if !report.Passed {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "core certification: failed (%d suites)\n", len(report.Suites))
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "core certification: passed (%d suites)\n", len(report.Suites))
				}
			}
			if err != nil {
				return apperr.Wrap(apperr.Internal, "conformance run core", "", err)
			}
			return nil
		},
	}
	run.Flags().BoolVar(&asJSON, "json", false, "emit certification report as JSON")
	run.Flags().StringVar(&filter, "filter", "", "run only suites matching id prefix")
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a certification matrix",
	}
	cmd.AddCommand(run)
	return cmd
}
