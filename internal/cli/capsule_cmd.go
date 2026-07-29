package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newCapsuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capsule",
		Short: "Create and restore portable dependency capsules",
	}
	cmd.AddCommand(newCapsuleCreateCmd())
	cmd.AddCommand(newCapsuleRestoreCmd())
	return cmd
}

func newCapsuleCreateCmd() *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Export lock, manifests, and cached blobs to a capsule archive",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "capsule.create", "", "missing app context")
			}
			res, err := app.CreateCapsule(cmd.Context(), ac, app.CapsuleCreateOptions{
				OutputPath: outputPath,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), app.FormatCapsuleCreateLine(res))
			return err
		},
	}
	cmd.Flags().StringVar(&outputPath, "output", "", "capsule archive path (default ./mew.capsule)")
	return cmd
}

func newCapsuleRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <path>",
		Short: "Import a capsule archive and run a frozen install",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "capsule.restore", "", "missing app context")
			}
			res, err := app.RestoreCapsule(cmd.Context(), ac, app.CapsuleRestoreOptions{
				ArchivePath: args[0],
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), app.FormatInstallSummary(res))
			return err
		},
	}
	return cmd
}
