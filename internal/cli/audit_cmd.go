package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/advisory"
	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newAuditCmd() *cobra.Command {
	var (
		asJSON bool
		fix    bool
	)
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit dependencies for known vulnerabilities",
		Long:  "Read-only scan of the lock graph against the cached OSV advisory database. Use --fix to print suggested safe version bumps (no manifest changes).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "audit", "", "missing app context")
			}
			result, err := app.Audit(cmd.Context(), ac, app.AuditOptions{Fix: fix})
			if err != nil {
				return err
			}
			if asJSON {
				payload := struct {
					advisory.AuditReport
					Fixes []advisory.FixSuggestion `json:"fixes,omitempty"`
				}{
					AuditReport: result.Report,
					Fixes:       result.Fixes,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(payload)
			}
			if text := advisory.FormatTable(result.Report); text != "" {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), text); err != nil {
					return err
				}
			}
			if fix {
				if fixes := advisory.FormatFixSuggestions(result.Fixes); fixes != "" {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), fixes); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print audit report as JSON")
	cmd.Flags().BoolVar(&fix, "fix", false, "print suggested safe version bumps (no write)")
	return cmd
}
