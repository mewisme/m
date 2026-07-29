package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newSBOMCmd() *cobra.Command {
	var (
		format         string
		redactInternal bool
		redactPattern  string
	)
	cmd := &cobra.Command{
		Use:   "sbom",
		Short: "Export a software bill of materials",
		Long:  "Emit a CycloneDX or SPDX SBOM from the project lock graph.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "sbom", "", "missing app context")
			}
			out, err := app.ExportSBOM(cmd.Context(), ac, app.ExportSBOMOptions{
				Format:         app.SBOMFormat(format),
				RedactInternal: redactInternal,
				RedactPattern:  redactPattern,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), string(out))
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "cyclonedx", "output format: cyclonedx or spdx")
	cmd.Flags().BoolVar(&redactInternal, "redact-internal", false, "redact scoped/internal package names")
	cmd.Flags().StringVar(&redactPattern, "redact-pattern", "", "regex of package names to redact")
	return cmd
}
