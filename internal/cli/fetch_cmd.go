package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newFetchCmd() *cobra.Command {
	var (
		planFile string
		destDir  string
		asJSON   bool
	)
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Download and extract packages from a fetch plan",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "fetch", "", "missing app context")
			}
			if planFile == "" {
				return apperr.New(apperr.Usage, "fetch", "", "--plan-file is required")
			}
			plan, err := app.LoadFetchPlan(planFile)
			if err != nil {
				return err
			}
			results, err := app.Fetch(cmd.Context(), ac, plan, destDir)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}
			for _, r := range results {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s@%s â†’ %s\n", r.Name, r.Version, r.Dest)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&planFile, "plan-file", "", "JSON fetch plan")
	cmd.Flags().StringVar(&destDir, "dir", ".", "destination directory for extracted trees")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print results as JSON")
	return cmd
}
