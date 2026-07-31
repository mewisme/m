package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify package attestations",
	}
	cmd.AddCommand(newVerifyProvenanceCmd())
	return cmd
}

func newVerifyProvenanceCmd() *cobra.Command {
	var attestationPath string
	cmd := &cobra.Command{
		Use:   "provenance [<pkg>]",
		Short: "Verify npm provenance attestation for a package",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "verify.provenance", "", "missing app context")
			}
			var pkgKey string
			if len(args) > 0 {
				pkgKey = args[0]
			}
			if pkgKey == "" && attestationPath == "" {
				return apperr.New(apperr.Usage, "verify.provenance", "", "package key or --attestation required")
			}
			res, err := app.VerifyProvenance(cmd.Context(), ac, app.VerifyProvenanceOptions{
				PackageKey:      pkgKey,
				AttestationPath: attestationPath,
			})
			if err != nil {
				return err
			}
			subject := res.SubjectPURL
			if subject == "" {
				subject = res.PackageName + "@" + res.PackageVersion
			}
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			return writeStaticOut(cmd, r.Summary(verifyProvenanceSummary(subject, res.DigestAlgo, res.DigestHex)))
		},
	}
	cmd.Flags().StringVar(&attestationPath, "attestation", "", "path to attestation JSON")
	return cmd
}
