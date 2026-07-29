package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newPublishCmd() *cobra.Command {
	var (
		dryRun          bool
		tag             string
		access          string
		otp             string
		provenance      bool
		packDestination string
		asJSON          bool
	)
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish a package",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "publish", "", "missing app context")
			}
			var tarball string
			if len(args) > 0 {
				tarball = args[0]
			}
			res, err := app.Publish(cmd.Context(), ac, app.PublishOptions{
				TarballPath:     tarball,
				DryRun:          dryRun,
				Tag:             tag,
				Access:          access,
				OTP:             otp,
				Provenance:      provenance,
				PackDestination: packDestination,
			})
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			if dryRun {
				_, err = fmt.Fprint(cmd.OutOrStdout(), app.FormatPublishPlan(res))
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), app.FormatPublishSuccess(res))
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and print plan without PUT")
	cmd.Flags().StringVar(&tag, "tag", "latest", "dist-tag for the published version")
	cmd.Flags().StringVar(&access, "access", "", "access for scoped packages: public or restricted")
	cmd.Flags().StringVar(&otp, "otp", "", "one-time password for registry 2FA")
	cmd.Flags().BoolVar(&provenance, "provenance", false, "request provenance attestation (hook only)")
	cmd.Flags().StringVar(&packDestination, "pack-destination", "", "pack output directory when no tarball is given")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	return cmd
}
