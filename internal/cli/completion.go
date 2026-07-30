package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/project"
)

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Long:      "Generate completion scripts for bash, zsh, fish, or powershell. Write the output to your shell's completion directory or evaluate it from the profile.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := strings.ToLower(args[0])
			out := cmd.OutOrStdout()
			switch shell {
			case "bash":
				return root.GenBashCompletion(out)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(out)
			default:
				return apperr.New(apperr.Usage, "completion", shell,
					fmt.Sprintf("unsupported shell %q (want bash|zsh|fish|powershell)", shell))
			}
		},
	}
}

func scriptNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return scriptNamesForCompletion(cmd, toComplete, false)
}

func rootScriptCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if !DirectScriptsEnabled(completionEffectiveConfig(cmd)) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return scriptNamesForCompletion(cmd, toComplete, true)
}

func completionEffectiveConfig(cmd *cobra.Command) *config.Effective {
	if ac := app.FromContext(cmd.Context()); ac != nil && ac.Config != nil {
		return ac.Config
	}
	return nil
}

func scriptNamesForCompletion(cmd *cobra.Command, toComplete string, excludeReserved bool) ([]string, cobra.ShellCompDirective) {
	cwd := ""
	if g := ownerFlags(cmd.Root()); g != nil && g.cwd != "" {
		cwd = g.cwd
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	root, err := project.FindRoot(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	doc, err := manifest.LoadCached(root)
	if err != nil || doc == nil || len(doc.Scripts) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	reserved := map[string]struct{}{}
	if excludeReserved {
		for _, n := range reservedFromRoot(cmd.Root()) {
			reserved[n] = struct{}{}
		}
	}
	names := make([]string, 0, len(doc.Scripts))
	for name := range doc.Scripts {
		if excludeReserved {
			if _, ok := reserved[name]; ok {
				continue
			}
		}
		if strings.HasPrefix(name, toComplete) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}
