package features

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
)

// FormatJSON returns deterministic public JSON for CLI output.
func FormatJSON(features []Feature) ([]byte, error) {
	public := PublicView(features)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(public); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FormatTable returns a fixed-width table for CLI output.
func FormatTable(features []Feature) string {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tMODULE\tFEATURE\tNUB\tMEW\tCLASS\tMVP")
	for _, f := range features {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			f.ID, f.Module, f.Name, f.NubStatus, f.MewStatus, f.CompatibilityClass, f.PrimaryMVP)
	}
	_ = w.Flush()
	return buf.String()
}

// Modules returns sorted unique module names.
func Modules(features []Feature) []string {
	seen := make(map[string]struct{})
	for _, f := range features {
		seen[f.Module] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for m := range seen {
		out = append(out, m)
	}
	sortStrings(out)
	return out
}

func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] < ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}
