package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/resolver"
)

func newResolveCmd() *cobra.Command {
	var (
		asJSON bool
		trace  bool
		plan   bool
	)
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve dependencies without installing",
		Long:  "Dry-run dependency resolution. Emits a canonical graph and optional decision trace. Full `m explain` arrives in MVP 0028; use --trace here.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = plan // dry resolve is the only mode in 0013
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "resolve", "", "missing app context")
			}
			proj, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			eng, err := resolver.NewFromApp(ac.Config, proj)
			if err != nil {
				return err
			}
			res, err := eng.Resolve(cmd.Context(), proj.Root, resolver.ResolveOptions{})
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			graph := res.Graph
			arrow := r.Settings().Symbols.Arrow
			if err := writeStaticOut(cmd, r.Summary(presentation.Summary{
				Status: presentation.StatusSuccess,
				Title:  "resolved",
				Metrics: []presentation.KeyValue{
					{Key: "packages", Value: strconv.Itoa(len(graph.Packages)), Style: presentation.ValueNumber},
					{Key: "edges", Value: strconv.Itoa(len(graph.Edges)), Style: presentation.ValueNumber},
					{Key: "decisions", Value: strconv.Itoa(len(res.Decisions)), Style: presentation.ValueNumber},
				},
			})); err != nil {
				return err
			}
			if trace {
				for _, d := range res.Decisions {
					line := fmt.Sprintf("%s@%s %s %s (%s)", d.Package, d.Requested, arrow, d.Selected, d.Reason)
					if len(d.PeerProviders) > 0 {
						line += fmt.Sprintf(" peerProviders=%v", d.PeerProviders)
					}
					if d.OverrideFrom != "" {
						line += fmt.Sprintf(" override=%q", d.OverrideFrom)
					}
					if len(d.Rejected) > 0 {
						line += fmt.Sprintf(" rejected=%v", d.Rejected)
					}
					if err := writeStaticOut(cmd, r.PlainText(line)); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&plan, "plan", true, "dry resolve without disk mutation (default true)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print Resolution as JSON")
	cmd.Flags().BoolVar(&trace, "trace", false, "print decision trace lines")
	return cmd
}
