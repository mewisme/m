package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/conformance"
)

func newConformanceRunRunnerCmd() *cobra.Command {
	var (
		asJSON  bool
		output  string
		force   bool
		groups  []string
		filters []string
	)
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "Run the runner certification matrix",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := findModuleRoot()
			if err != nil {
				return apperr.Wrap(apperr.Internal, "conformance run runner", "", err)
			}
			if output != "" && !force {
				if _, err := os.Stat(output); err == nil {
					return apperr.New(apperr.Usage, "conformance run runner", output, "output file exists (use --force)")
				}
			}
			report, err := conformance.RunRunner(cmd.Context(), conformance.RunnerRunOptions{
				RepoRoot: repoRoot,
				Groups:   groups,
				Filters:  filters,
			})
			if output != "" {
				data, encErr := report.EncodeRunnerReportJSON()
				if encErr != nil {
					return apperr.Wrap(apperr.IO, "conformance run runner", output, encErr)
				}
				if writeErr := writeFileAtomic(output, data, force); writeErr != nil {
					return apperr.Wrap(apperr.IO, "conformance run runner", output, writeErr)
				}
			}
			if asJSON {
				data, encErr := report.EncodeRunnerReportJSON()
				if encErr != nil {
					return apperr.Wrap(apperr.Internal, "conformance run runner", "", encErr)
				}
				if _, writeErr := cmd.OutOrStdout().Write(data); writeErr != nil {
					return writeErr
				}
				_, _ = cmd.OutOrStdout().Write([]byte("\n"))
			} else {
				for _, s := range report.Suites {
					line := fmt.Sprintf("%s\t%s", s.ID, s.Result)
					if s.FailureReason != "" {
						line += "\t" + s.FailureReason
					}
					if _, writeErr := fmt.Fprintln(cmd.OutOrStdout(), line); writeErr != nil {
						return writeErr
					}
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "runner certification: %s (%d suites)\n", report.Overall, len(report.Suites))
			}
			if err != nil {
				return apperr.Wrap(apperr.Internal, "conformance run runner", "", err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit certification report as JSON")
	cmd.Flags().StringVar(&output, "output", "", "write report JSON to path")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing --output file")
	cmd.Flags().StringArrayVar(&groups, "group", nil, "run suites in group (repeatable)")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "run exact suite id (repeatable)")
	return cmd
}

func newConformanceVerifyRunnerCmd() *cobra.Command {
	var (
		reports []string
		output  string
		force   bool
	)
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "Aggregate per-platform runner certification reports",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "" && !force {
				if _, err := os.Stat(output); err == nil {
					return apperr.New(apperr.Usage, "conformance verify runner", output, "output file exists (use --force)")
				}
			}
			summary, err := conformance.VerifyRunnerReports(conformance.RunnerVerifyOptions{
				ReportPaths: reports,
				OutputPath:  output,
			})
			data, encErr := json.MarshalIndent(summary, "", "  ")
			if encErr != nil {
				return apperr.Wrap(apperr.Internal, "conformance verify runner", "", encErr)
			}
			if _, writeErr := cmd.OutOrStdout().Write(data); writeErr != nil {
				return writeErr
			}
			_, _ = cmd.OutOrStdout().Write([]byte("\n"))
			if err != nil {
				return apperr.Wrap(apperr.Internal, "conformance verify runner", "", err)
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&reports, "report", nil, "per-platform runner report path (repeatable)")
	cmd.Flags().StringVar(&output, "output", "", "write certification summary JSON to path")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing --output file")
	return cmd
}

func writeFileAtomic(path string, data []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file exists")
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
