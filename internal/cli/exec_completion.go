package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// execCompletionState tracks parser position for m exec completion.
type execCompletionState int

const (
	execStateFlags execCompletionState = iota
	execStatePackageValue
	execStateBeforeSelector
	execStateDone
)

func execCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	state, _ := execParserState(args)
	switch state {
	case execStateFlags:
		return []string{"--package"}, cobra.ShellCompDirectiveNoFileComp
	case execStatePackageValue, execStateBeforeSelector:
		return nil, cobra.ShellCompDirectiveNoFileComp
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func execParserState(args []string) (execCompletionState, int) {
	expectPackage := false
	for i, arg := range args {
		if arg == "--" {
			return execStateDone, i
		}
		if expectPackage {
			expectPackage = false
			continue
		}
		if arg == "--package" {
			expectPackage = true
			continue
		}
		if strings.HasPrefix(arg, "--package=") {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return execStateDone, i
	}
	if expectPackage {
		return execStatePackageValue, len(args)
	}
	if len(args) == 0 {
		return execStateFlags, 0
	}
	return execStateBeforeSelector, len(args)
}
