package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newPackCmd() *cobra.Command {
	var packDestination string
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Create a package tarball",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "pack", "", "missing app context")
			}
			var pkgDir string
			if len(args) > 0 {
				pkgDir = args[0]
			}
			res, err := app.Pack(cmd.Context(), ac, app.PackOptions{
				PackDestination: packDestination,
				PackageDir:      pkgDir,
			})
			if err != nil {
				return err
			}
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			return writeStaticPrint(cmd, r.PlainText(app.FormatPackLine(res)))
		},
	}
	cmd.Flags().StringVar(&packDestination, "pack-destination", "", "directory for the .tgz (default cwd)")
	return cmd
}
