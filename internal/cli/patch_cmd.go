package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/presentation"
)

func newPatchCmd() *cobra.Command {
	var (
		editDir   string
		commit    bool
		pnpmMajor int
	)
	cmd := &cobra.Command{
		Use:   "patch <package>",
		Short: "Extract a dependency for patching or commit a patch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "patch", "", "missing app context")
			}
			result, err := app.Patch(cmd.Context(), ac, app.PatchOptions{
				Package:   args[0],
				EditDir:   editDir,
				Commit:    commit,
				PnpmMajor: pnpmMajor,
			})
			if err != nil {
				return err
			}
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			if commit {
				return writeStaticOut(cmd, r.Status(presentation.StatusLine{
					Status: presentation.StatusSuccess,
					Text:   fmt.Sprintf("wrote %s", result.PatchPath),
				}))
			}
			return writeStaticOut(cmd, r.Status(presentation.StatusLine{
				Status: presentation.StatusInfo,
				Text:   result.EditDir,
			}))
		},
	}
	cmd.Flags().StringVar(&editDir, "edit-dir", "", "directory to extract the package for editing")
	cmd.Flags().BoolVar(&commit, "commit", false, "write patches/*.patch, update patchedDependencies, and install")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")
	return cmd
}
