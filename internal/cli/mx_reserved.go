package cli

import (
	"github.com/spf13/cobra"
)

// MXReservedNames returns built-in mx commands that preempt DLX dispatch.
func MXReservedNames(root *cobra.Command) []string {
	if root == nil {
		return []string{"version", "completion", "cache"}
	}
	names := []string{}
	for _, c := range root.Commands() {
		if c == nil || c.Hidden {
			continue
		}
		names = append(names, c.Name())
		for _, a := range c.Aliases {
			names = append(names, a)
		}
	}
	return names
}

// IsMXReserved reports whether selector is a built-in mx command.
func IsMXReserved(root *cobra.Command, selector string) bool {
	for _, name := range MXReservedNames(root) {
		if name == selector {
			return true
		}
	}
	return false
}
