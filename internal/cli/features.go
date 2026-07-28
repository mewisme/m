package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/features"
)

func newFeaturesCmd() *cobra.Command {
	var (
		format string
		module string
		status string
	)

	cmd := &cobra.Command{
		Use:   "features",
		Short: "List capability inventory and parity status",
		Long:  "Show the machine-readable feature inventory as a table or JSON document.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unexpected arguments: %v", args)
			}
			return runFeatures(cmd, format, module, status)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format: table or json")
	cmd.Flags().StringVar(&module, "module", "", "filter by module name")
	cmd.Flags().StringVar(&status, "status", "", "filter by mew_status")

	return cmd
}

func runFeatures(cmd *cobra.Command, format, module, status string) error {
	if format != "table" && format != "json" {
		return fmt.Errorf("unsupported format %q (want table or json)", format)
	}

	inv, err := features.LoadEmbedded()
	if err != nil {
		return err
	}
	filtered := inv.Filter(module, status)
	out := cmd.OutOrStdout()

	switch format {
	case "json":
		b, err := features.FormatJSON(filtered)
		if err != nil {
			return err
		}
		_, err = out.Write(b)
		return err
	case "table":
		_, err := fmt.Fprint(out, features.FormatTable(filtered))
		return err
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}
