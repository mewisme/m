package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/policy"
)

func newPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Evaluate organizational supply-chain policy",
	}
	cmd.AddCommand(newPolicyCheckCmd())
	return cmd
}

func newPolicyCheckCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check dependencies against org policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "policy.check", "", "missing app context")
			}
			result, err := app.CheckPolicy(cmd.Context(), ac)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				if err := enc.Encode(result); err != nil {
					return err
				}
				if !result.Passed {
					return apperr.New(apperr.Policy, "policy.check", "", "org policy violations found")
				}
				return nil
			}
			return formatPolicyResult(result, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print policy result as JSON")
	return cmd
}

func formatPolicyResult(result policy.PolicyResult, w interface{ Write([]byte) (int, error) }) error {
	if result.Passed && len(result.Violations) == 0 {
		_, err := fmt.Fprintln(w, "policy check passed")
		return err
	}
	for _, v := range result.Violations {
		line := fmt.Sprintf("%s %s: %s", v.Severity, v.Package, v.Message)
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	if !result.Passed {
		return apperr.New(apperr.Policy, "policy.check", "", "org policy violations found")
	}
	return nil
}
