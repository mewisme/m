package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/lifecycle"
)

func newTrustCmd() *cobra.Command {
	var interactive bool
	cmd := &cobra.Command{
		Use:   "trust [package...]",
		Short: "Trust lifecycle scripts for packages",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "trust", "", "missing app context")
			}
			store, err := lifecycle.LoadTrust(ac.CWD)
			if err != nil {
				return err
			}
			if interactive {
				return runTrustInteractive(cmd, store, args)
			}
			if len(args) == 0 {
				return apperr.New(apperr.Usage, "trust", "", "package name required (or use --interactive)")
			}
			for _, pkg := range args {
				if err := store.AddTrusted(pkg); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "trusted %s\n", pkg)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&interactive, "interactive", false, "prompt to trust each package name")
	return cmd
}

func newApproveBuildsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "approve-builds [package...]",
		Short:   "Approve lifecycle scripts for packages (alias for m trust)",
		Aliases: []string{"approve-build"},
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "approve-builds", "", "missing app context")
			}
			if len(args) == 0 {
				return apperr.New(apperr.Usage, "approve-builds", "", "package name required")
			}
			store, err := lifecycle.LoadTrust(ac.CWD)
			if err != nil {
				return err
			}
			for _, pkg := range args {
				if err := store.AddTrusted(pkg); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "trusted %s\n", pkg)
			}
			return nil
		},
	}
	return cmd
}

func runTrustInteractive(cmd *cobra.Command, store *lifecycle.TrustStore, seeds []string) error {
	names := append([]string(nil), seeds...)
	if len(names) == 0 {
		return apperr.New(apperr.Usage, "trust", "", "provide package names with --interactive")
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	for _, pkg := range names {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Trust lifecycle scripts for %s? [y/N] ", pkg)
		line, err := reader.ReadString('\n')
		if err != nil {
			return apperr.Wrap(apperr.IO, "trust", pkg, err)
		}
		if strings.EqualFold(strings.TrimSpace(line), "y") {
			if err := store.AddTrusted(pkg); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "trusted %s\n", pkg)
		}
	}
	return nil
}
