package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
)

func newInstallCmd() *cobra.Command {
	var (
		prod        bool
		frozen      bool
		dryRun      bool
		keepJournal bool
		linkerMode  string
		asJSON      bool
	)
	cmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i"},
		Short:   "Install dependencies",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "install", "", "missing app context")
			}
			opts := app.InstallOptions{Prod: prod, Frozen: frozen, DryRun: dryRun, KeepJournal: keepJournal, Linker: linkerMode}
			result, err := app.Install(cmd.Context(), ac, opts)
			outErr := writeInstallResult(cmd, result, asJSON, dryRun)
			if err != nil {
				if outErr != nil {
					return outErr
				}
				return err
			}
			return outErr
		},
	}
	cmd.Flags().BoolVar(&prod, "prod", false, "omit devDependencies")
	cmd.Flags().BoolVar(&frozen, "frozen-lockfile", false, "fail if package.json and m.lock drift")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and print plan without mutating disk")
	cmd.Flags().BoolVar(&keepJournal, "journal", false, "keep transaction journal after success")
	cmd.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	return cmd
}

func newAddCmd() *cobra.Command {
	var (
		dev        bool
		saveExact  bool
		linkerMode string
		asJSON     bool
	)
	cmd := &cobra.Command{
		Use:   "add <package>",
		Short: "Add a dependency",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "add", "", "missing app context")
			}
			result, err := app.Add(cmd.Context(), ac, args[0], app.AddOptions{
				Dev:       dev,
				SaveExact: saveExact,
				Install:   app.InstallOptions{Linker: linkerMode},
			})
			outErr := writeInstallResult(cmd, result, asJSON, false)
			if err != nil {
				if outErr != nil {
					return outErr
				}
				return err
			}
			return outErr
		},
	}
	cmd.Flags().BoolVarP(&dev, "save-dev", "D", false, "save to devDependencies")
	cmd.Flags().BoolVarP(&saveExact, "save-exact", "E", false, "save exact version")
	cmd.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	var (
		linkerMode string
		asJSON     bool
	)
	cmd := &cobra.Command{
		Use:     "remove <package>",
		Aliases: []string{"rm"},
		Short:   "Remove a dependency",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "remove", "", "missing app context")
			}
			result, err := app.Remove(cmd.Context(), ac, args[0], app.InstallOptions{Linker: linkerMode})
			outErr := writeInstallResult(cmd, result, asJSON, false)
			if err != nil {
				if outErr != nil {
					return outErr
				}
				return err
			}
			return outErr
		},
	}
	cmd.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	return cmd
}

func newCiCmd() *cobra.Command {
	var (
		prod       bool
		linkerMode string
		asJSON     bool
	)
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "Clean install from lockfile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "ci", "", "missing app context")
			}
			result, err := app.Install(cmd.Context(), ac, app.InstallOptions{
				Prod:   prod,
				Frozen: true,
				Linker: linkerMode,
			})
			outErr := writeInstallResult(cmd, result, asJSON, false)
			if err != nil {
				if outErr != nil {
					return outErr
				}
				return err
			}
			return outErr
		},
	}
	cmd.Flags().BoolVar(&prod, "prod", false, "omit devDependencies")
	cmd.Flags().StringVar(&linkerMode, "linker", "", "node linker mode: hoisted or isolated")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print result as JSON")
	return cmd
}

func writeInstallResult(cmd *cobra.Command, result app.InstallResult, asJSON, dryRun bool) error {
	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	prefix := ""
	if dryRun {
		prefix = "dry-run: "
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", prefix, app.FormatInstallSummary(result))
	return err
}
