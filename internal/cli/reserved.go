package cli

import (
	"sort"
)

// stubSpec describes a reserved not-yet-implemented command.
type stubSpec struct {
	Use     string
	Aliases []string
	MVP     string
	Short   string
}

var stubCommands = []stubSpec{
	{Use: "run", MVP: "0040", Short: "Run a package script"},
	{Use: "exec", MVP: "0043", Short: "Execute a local package binary"},
	{Use: "init", MVP: "0070", Short: "Initialize a project"},
	{Use: "link", MVP: "0026", Short: "Link a local package"},
}

// shippedBuiltins are always reserved (implemented today).
var shippedBuiltins = []string{
	"version", "features", "development", "config", "project", "pkg", "cache", "store", "view", "resolve", "fetch", "lock", "install", "i", "add", "remove", "rm", "ci", "update", "patch", "explain", "plan", "snapshot", "history", "recover", "rollback", "diff", "pack", "capsule", "publish", "doctor", "bench", "benchmark", "conformance", "audit", "sbom", "policy", "verify", "completion", "__dispatch", "help",
}

// ReservedNames returns sorted command names that cannot be shadowed by scripts (0042).
func ReservedNames() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range shippedBuiltins {
		add(s)
	}
	for _, st := range stubCommands {
		add(st.Use)
		for _, a := range st.Aliases {
			add(a)
		}
	}
	sort.Strings(out)
	return out
}

// IsReserved reports whether name is a reserved built-in or stub.
func IsReserved(name string) bool {
	for _, n := range ReservedNames() {
		if n == name {
			return true
		}
	}
	return false
}
