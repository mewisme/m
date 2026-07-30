package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/features"
	"github.com/mewisme/mew/internal/presentation"
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
		g := ownerFlags(cmd.Root())
		r := g.mustStaticRenderer(cmd, nil)
		_, err := fmt.Fprintln(out, r.Table(featuresTableModel(filtered)))
		return err
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func featuresTableModel(rows []features.Feature) presentation.TableModel {
	cols := []presentation.TableColumn{
		{Key: "id", Header: "ID", MinWidth: 8, Prefer: 28, Primary: true, Truncate: presentation.TruncateMiddle},
		{Key: "module", Header: "MODULE", MinWidth: 6, Prefer: 16, Truncate: presentation.TruncateMiddle},
		{Key: "feature", Header: "FEATURE", MinWidth: 8, Prefer: 28, Truncate: presentation.TruncateMiddle},
		{Key: "nub", Header: "NUB", MinWidth: 4, Prefer: 12},
		{Key: "mew", Header: "MEW", MinWidth: 4, Prefer: 12},
		{Key: "class", Header: "CLASS", MinWidth: 4, Prefer: 12},
		{Key: "mvp", Header: "MVP", MinWidth: 4, Prefer: 10},
	}
	tableRows := make([]map[string]string, 0, len(rows))
	for _, f := range rows {
		tableRows = append(tableRows, map[string]string{
			"id":      f.ID,
			"module":  f.Module,
			"feature": f.Name,
			"nub":     string(f.NubStatus),
			"mew":     string(f.MewStatus),
			"class":   string(f.CompatibilityClass),
			"mvp":     f.PrimaryMVP,
		})
	}
	return presentation.TableModel{Columns: cols, Rows: tableRows}
}
