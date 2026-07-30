package cli

import (
	"sort"

	"github.com/spf13/cobra"
)

// reservedFromRoot enumerates command names and aliases that direct scripts must not shadow.
func reservedFromRoot(root *cobra.Command) []string {
	if root == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd == nil || cmd == root {
			for _, c := range cmd.Commands() {
				walk(c)
			}
			return
		}
		if cmd.Name() != "" {
			if _, ok := seen[cmd.Name()]; !ok {
				seen[cmd.Name()] = struct{}{}
				out = append(out, cmd.Name())
			}
		}
		for _, a := range cmd.Aliases {
			if a == "" {
				continue
			}
			if _, ok := seen[a]; !ok {
				seen[a] = struct{}{}
				out = append(out, a)
			}
		}
		for _, c := range cmd.Commands() {
			walk(c)
		}
	}
	walk(root)
	// Cobra handles help via meta flags; reserve the name for direct-script collision.
	if _, ok := seen["help"]; !ok {
		out = append(out, "help")
	}
	sort.Strings(out)
	return out
}

// reservedSetForRoot builds a lookup set from the Cobra tree (cached per root pointer).
func reservedSetForRoot(root *cobra.Command) map[string]struct{} {
	names := reservedFromRoot(root)
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	return set
}

// driftAgainstShippedBuiltins returns names present in shippedBuiltins but missing from the tree.
func driftAgainstShippedBuiltins(root *cobra.Command) []string {
	tree := reservedSetForRoot(root)
	var missing []string
	for _, name := range shippedBuiltins {
		if name == "help" {
			continue // default help command is disabled on the m root
		}
		if _, ok := tree[name]; !ok {
			missing = append(missing, name)
		}
	}
	for _, st := range stubCommands {
		if _, ok := tree[st.Use]; !ok {
			missing = append(missing, st.Use)
		}
		for _, a := range st.Aliases {
			if a == "" {
				continue
			}
			if _, ok := tree[a]; !ok {
				missing = append(missing, a)
			}
		}
	}
	sort.Strings(missing)
	return missing
}
