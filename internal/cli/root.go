package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
)

// NewMRoot returns the root Cobra command for the m binary.
func NewMRoot(info BuildInfo) *cobra.Command {
	invoked := InvokedBinary(os.Args[0], "m")
	use := DisplayName(invoked)
	if use != "m" && use != "mew" {
		use = "m"
	}
	root := &cobra.Command{
		Use:   use,
		Short: "Mew — JavaScript toolchain and package manager",
		Long:  "Mew is a Go implementation of the Nub product model for package management, scripts, and runtime augmentation.",
	}
	root.Version = info.Version
	root.CompletionOptions.DisableDefaultCmd = true
	root.DisableAutoGenTag = true
	root.SilenceUsage = true
	root.SilenceErrors = true
	g := attachGlobals(root)
	attachAppPreRun(root, g, info)

	versionLabel := use
	root.AddCommand(newVersionCmd(versionLabel, info))
	root.AddCommand(newFeaturesCmd())
	root.AddCommand(newDevelopmentCmd())
	root.AddCommand(newConfigCmd(g))
	root.AddCommand(newProjectCmd())
	root.AddCommand(newPkgCmd())
	root.AddCommand(newCacheCmd())
	root.AddCommand(newStoreCmd())
	root.AddCommand(newViewCmd())
	root.AddCommand(newResolveCmd())
	root.AddCommand(newFetchCmd())
	root.AddCommand(newLockCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newAddCmd())
	root.AddCommand(newRemoveCmd())
	root.AddCommand(newCiCmd())
	root.AddCommand(newSnapshotCmd())
	root.AddCommand(newRecoverCmd())
	root.AddCommand(newRollbackCmd())
	root.AddCommand(newCompletionCmd(root))
	root.AddCommand(newDispatchCmd(root))
	registerStubs(root)

	return root
}

// NewMXRoot returns the root Cobra command for the mx binary.
func NewMXRoot(info BuildInfo) *cobra.Command {
	invoked := InvokedBinary(os.Args[0], "mx")
	use := DisplayName(invoked)
	if use != "mx" && use != "mewx" {
		use = "mx"
	}
	root := &cobra.Command{
		Use:   use,
		Short: "Mewx — package executable runner",
		Long:  "Mewx executes local or temporary package binaries.",
	}
	root.Version = info.Version
	root.CompletionOptions.DisableDefaultCmd = true
	root.DisableAutoGenTag = true
	root.SilenceUsage = true
	root.SilenceErrors = true
	g := attachGlobals(root)
	attachAppPreRun(root, g, info)

	root.AddCommand(newVersionCmd(use, info))
	root.AddCommand(newCompletionCmd(root))

	return root
}

// ExecuteM runs the m command tree.
func ExecuteM(info BuildInfo) int {
	return execute(NewMRoot(info))
}

// ExecuteMX runs the mx command tree.
func ExecuteMX(info BuildInfo) int {
	return execute(NewMXRoot(info))
}

func attachAppPreRun(root *cobra.Command, g *globalFlags, info BuildInfo) {
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		cwd := g.cwd
		if cwd == "" {
			cwd, _ = os.Getwd()
		} else {
			abs, err := filepath.Abs(cwd)
			if err != nil {
				return err
			}
			cwd = abs
		}
		ac, err := app.New(ctx, app.Options{
			CWD:           cwd,
			ConfigPath:    g.configPath,
			Offline:       g.offline,
			PreferOffline: g.preferOffline,
			Reporter:      g.newReporter(),
			Version:       info.Version,
			Commit:        info.Commit,
			BuildDate:     info.BuildDate,
		})
		if err != nil {
			return err
		}
		cmd.SetContext(app.WithContext(ctx, ac))
		return nil
	}
}

func newVersionCmd(binary string, info BuildInfo) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				doc := map[string]string{
					"binary":    binary,
					"version":   info.Version,
					"commit":    info.Commit,
					"buildDate": info.BuildDate,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(doc)
			}
			line := fmt.Sprintf("%s %s", binary, info.Version)
			if info.Commit != "" {
				line = fmt.Sprintf("%s (%s)", line, info.Commit)
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), line); err != nil {
				return err
			}
			if info.BuildDate != "" {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "built %s\n", info.BuildDate)
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print version as JSON")
	return cmd
}
