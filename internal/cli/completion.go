package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/apperr"
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
