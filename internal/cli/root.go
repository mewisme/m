package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/presentation"
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
		Short: "MewJS — JavaScript toolchain and package manager",
		Long:  "MewJS (Mew) is a Go-based JavaScript toolchain and package manager for package management, scripts, and runtime augmentation.",
	}
	root.Version = info.Version
	root.CompletionOptions.DisableDefaultCmd = true
	root.DisableAutoGenTag = true
	root.SilenceUsage = true
	root.SilenceErrors = true
	g := attachGlobals(root)
	g.bindRecursive(root)
	attachAppPreRun(root, g, info)
	storeRootBuildInfo(root, info)

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
	root.AddCommand(newLsCmd())
	root.AddCommand(newOutdatedCmd())
	root.AddCommand(newAddCmd())
	root.AddCommand(newRemoveCmd())
	root.AddCommand(newCiCmd())
	root.AddCommand(newDedupeCmd())
	root.AddCommand(newPruneCmd())
	root.AddCommand(newTrustCmd())
	root.AddCommand(newApproveBuildsCmd())
	root.AddCommand(newBuildsCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newPatchCmd())
	root.AddCommand(newExplainCmd())
	root.AddCommand(newPlanCmd())
	root.AddCommand(newSnapshotCmd())
	root.AddCommand(newHistoryCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newRecoverCmd())
	root.AddCommand(newRollbackCmd())
	root.AddCommand(newPackCmd())
	root.AddCommand(newCapsuleCmd())
	root.AddCommand(newPublishCmd())
	root.AddCommand(newVerifyCmd())
	root.AddCommand(newPMDoctorCmd())
	root.AddCommand(newBenchCmd())
	root.AddCommand(newConformanceCmd())
	root.AddCommand(newAuditCmd())
	root.AddCommand(newSBOMCmd())
	root.AddCommand(newPolicyCmd())
	root.AddCommand(newCompletionCmd(root))
	root.AddCommand(newDispatchCmd(root))
	root.AddCommand(newRunCmd())
	root.AddCommand(newExecCmd())
	root.AddCommand(newEnvCmd())
	registerStubs(root)
	root.ValidArgsFunction = rootScriptCompletion
	configureGroupedHelp(root)

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
		Short: "MewJS — package executable runner",
		Long:  "MewJS package executable runner. Executes local or temporary package binaries.",
	}
	root.Version = info.Version
	root.CompletionOptions.DisableDefaultCmd = true
	root.DisableAutoGenTag = true
	root.SilenceUsage = true
	root.SilenceErrors = true
	g := attachGlobals(root)
	attachAppPreRun(root, g, info)
	storeRootBuildInfo(root, info)

	root.AddCommand(newVersionCmd(use, info))
	root.AddCommand(newCompletionCmd(root))
	root.AddCommand(newMXCacheCmd())
	configureGroupedHelp(root)

	return root
}

// ExecuteM runs the m command tree.
func ExecuteM(info BuildInfo) int {
	return execute(NewMRoot(info), info, os.Args[1:])
}

// ExecuteMX runs the mx command tree.
func ExecuteMX(info BuildInfo) int {
	return execute(NewMXRoot(info), info, os.Args[1:])
}

func attachAppPreRun(root *cobra.Command, g *globalFlags, info BuildInfo) {
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := g.validateStructuredConflict(cmd); err != nil {
			return err
		}
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		ac, err := buildAppContext(ctx, cmd, g, info)
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
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd, nil)
			out := r.Status(presentation.StatusLine{
				Text: fmt.Sprintf("%s %s", binary, info.Version),
			})
			var kvs []presentation.KeyValue
			if info.Commit != "" {
				kvs = append(kvs, presentation.KeyValue{Key: "commit", Value: info.Commit})
			}
			if info.BuildDate != "" {
				kvs = append(kvs, presentation.KeyValue{Key: "buildDate", Value: info.BuildDate, Style: presentation.ValueMuted})
			}
			if len(kvs) > 0 {
				out += "\n" + r.KeyValues(kvs)
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), out)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print version as JSON")
	return cmd
}
