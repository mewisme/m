package dlx

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// InferModeABin selects a bin for Mode A from declared bin names.
func InferModeABin(packageName string, bins []string) (string, error) {
	if len(bins) == 0 {
		return "", apperr.New(apperr.NotFound, "dlx.bininfer", packageName, "package has no bin")
	}
	sort.Strings(bins)
	if len(bins) == 1 {
		return bins[0], nil
	}
	want := UnscopedName(packageName)
	for _, b := range bins {
		if b == want {
			return b, nil
		}
	}
	return "", apperr.New(apperr.Usage, "dlx.bininfer", packageName,
		fmt.Sprintf("ambiguous bins for %s: %s", packageName, strings.Join(bins, ", ")))
}

// ResolveModeBCommand resolves an explicit command across direct package bins.
func ResolveModeBCommand(command string, owners map[string]map[string]string) (owner string, err error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", apperr.New(apperr.Usage, "dlx.bininfer", "", "command required")
	}
	var matches []string
	for pkg, bins := range owners {
		for _, b := range BinNames(bins) {
			if b == command {
				matches = append(matches, pkg)
				break
			}
		}
	}
	if len(matches) == 0 {
		return "", apperr.New(apperr.NotFound, "dlx.bininfer", command, "command not found in requested packages")
	}
	sort.Strings(matches)
	if len(matches) > 1 {
		return "", multipleOwnerError(command, matches)
	}
	return matches[0], nil
}

func multipleOwnerError(command string, owners []string) error {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Command %q is exposed by multiple requested packages:\n", command))
	for _, o := range owners {
		b.WriteString("  ")
		b.WriteString(o)
		b.WriteByte('\n')
	}
	b.WriteString("\nUse a package set that exposes ")
	b.WriteString(command)
	b.WriteString(" unambiguously.")
	return apperr.New(apperr.Usage, "dlx.bininfer", command, b.String())
}

// BinNames extracts sorted bin command names from a map.
func BinNames(bins map[string]string) []string {
	return BinNamesMapValues(bins)
}

// BinNamesMapValues extracts sorted bin names from a map value.
func BinNamesMapValues(bins map[string]string) []string {
	out := make([]string, 0, len(bins))
	for name := range bins {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
