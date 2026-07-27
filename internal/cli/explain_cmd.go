package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/resolver"
)

func newExplainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain resolution decisions",
		Long:  "Peer conflict diagnostics ship in MVP 0020; full explain UX arrives in MVP 0028.",
	}
	cmd.AddCommand(newExplainPeerCmd())
	return cmd
}

func newExplainPeerCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "peer <name>",
		Short: "Explain an unsatisfied peer dependency",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "explain.peer", "", "missing app context")
			}
			proj, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			prior, err := app.ReadLockGraph(cmd.Context(), ac)
			if err != nil {
				if apperr.CodeOf(err) != apperr.IO {
					return err
				}
				prior = nil
			}
			eng, err := resolver.NewFromApp(ac.Config, proj, os.Environ())
			if err != nil {
				return err
			}
			tree, err := eng.ExplainPeer(cmd.Context(), proj.Root, args[0], resolver.ResolveOptions{
				Prior: prior,
				Hints: prior,
			})
			if err != nil {
				return err
			}
			if tree == nil {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "no peer conflict for %q\n", args[0])
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(tree)
			}
			return printConflictTree(cmd, *tree)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print conflict tree as JSON")
	return cmd
}

func printConflictTree(cmd *cobra.Command, tree resolver.ConflictTree) error {
	_, err := fmt.Fprint(cmd.OutOrStdout(), resolver.FormatConflictTree(tree))
	return err
}
