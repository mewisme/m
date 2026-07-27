package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
)

func newStoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store",
		Short: "Manage the content store",
	}
	cmd.AddCommand(newStorePathCmd())
	cmd.AddCommand(newStoreStatusCmd())
	cmd.AddCommand(newStorePruneCmd())
	return cmd
}

func newStorePathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the global content store directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "store.path", "", "missing app context")
			}
			root, err := config.StoreRoot(ac.Config)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), root)
			return err
		},
	}
}

func newStoreStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report store path, package count, and bytes on disk",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "store.status", "", "missing app context")
			}
			st, err := app.StoreStatusReport(cmd.Context(), ac)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(st)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), app.FormatStoreStatus(st))
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return cmd
}

func newStorePruneCmd() *cobra.Command {
	var dryRun, asJSON bool
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove store packages not referenced by any store manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "store.prune", "", "missing app context")
			}
			var roots []string
			if p, err := app.OpenProject(cmd.Context(), ac); err == nil {
				roots = app.DefaultStoreScanRoots(p.Root)
			} else {
				roots = app.DefaultStoreScanRoots("")
			}
			res, err := app.PruneStore(cmd.Context(), ac, dryRun, roots)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(res)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "removed=%d kept=%d dry_run=%v\n", res.Removed, res.Kept, res.DryRun)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report packages that would be removed without deleting")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	return cmd
}
