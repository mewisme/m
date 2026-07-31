package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lifecycle"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/prompt"
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
				return runTrustInteractive(cmd, ac, store, args)
			}
			if len(args) == 0 {
				return apperr.New(apperr.Usage, "trust", "", "package name required (or use --interactive)")
			}
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			for _, pkg := range args {
				if err := store.AddTrusted(pkg); err != nil {
					return err
				}
				if err := writeStaticErr(cmd, r.Status(presentation.StatusLine{
					Status: presentation.StatusSuccess,
					Text:   "trusted",
					Detail: pkg,
				})); err != nil {
					return err
				}
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
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			for _, pkg := range args {
				if err := store.AddTrusted(pkg); err != nil {
					return err
				}
				if err := writeStaticErr(cmd, r.Status(presentation.StatusLine{
					Status: presentation.StatusSuccess,
					Text:   "trusted",
					Detail: pkg,
				})); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

func runTrustInteractive(cmd *cobra.Command, ac *app.Context, store *lifecycle.TrustStore, seeds []string) error {
	names := append([]string(nil), seeds...)
	if len(names) == 0 {
		return apperr.New(apperr.Usage, "trust", "", "provide package names with --interactive")
	}
	if ac == nil || !ac.CanPrompt || ac.Prompter == nil {
		return apperr.New(apperr.Usage, "trust", "", "interactive trust requires a TTY on stdin")
	}
	g := ownerFlags(cmd.Root())
	r := g.mustStaticRenderer(cmd)
	for _, pkg := range names {
		ans, err := ac.Prompter.Prompt(cmd.Context(), prompt.PromptRequest{
			ID:        "trust.interactive",
			Kind:      prompt.PromptConfirm,
			Title:     fmt.Sprintf("Trust lifecycle scripts for %s?", pkg),
			Dangerous: true,
			DefaultID: prompt.OptionReject,
			Fields:    []prompt.Field{{Key: "Package", Value: pkg}},
			Options: []prompt.Option{
				{ID: prompt.OptionReject, Label: "No"},
				{ID: prompt.OptionApprove, Label: "Yes"},
			},
		})
		if err != nil {
			return err
		}
		if ans.Cancelled || ans.OptionID != prompt.OptionApprove {
			if err := writeStaticErr(cmd, r.Status(presentation.StatusLine{
				Status: presentation.StatusInfo,
				Text:   "skipped",
				Detail: pkg,
			})); err != nil {
				return err
			}
			continue
		}
		if err := store.AddTrusted(pkg); err != nil {
			return err
		}
		if err := writeStaticErr(cmd, r.Status(presentation.StatusLine{
			Status: presentation.StatusSuccess,
			Text:   "trusted",
			Detail: pkg,
		})); err != nil {
			return err
		}
	}
	return nil
}
