package cli

import (
	"sync"
)

// stubSpec describes a reserved not-yet-implemented command.
type stubSpec struct {
	Use     string
	Aliases []string
	MVP     string
	Short   string
}

var stubCommands = []stubSpec{
	{Use: "init", MVP: "0070", Short: "Initialize a project"},
	{Use: "link", MVP: "0026", Short: "Link a local package"},
}

// shippedBuiltins are always reserved (implemented today); drift-tested against the Cobra tree.
var shippedBuiltins = []string{
	"version", "features", "development", "config", "project", "pkg", "cache", "store", "view", "resolve", "fetch", "lock", "install", "i", "add", "remove", "rm", "ci", "update", "patch", "explain", "plan", "snapshot", "history", "recover", "rollback", "diff", "pack", "capsule", "publish", "doctor", "bench", "benchmark", "conformance", "audit", "sbom", "policy", "verify", "completion", "__dispatch", "run", "exec", "help",
}

var (
	reservedOnce  sync.Once
	reservedCache []string
)

func ensureReservedCache() {
	reservedOnce.Do(func() {
		reservedCache = reservedFromRoot(NewMRoot(BuildInfo{}))
	})
}

// ReservedNames returns sorted command names that cannot be shadowed by scripts (0042).
func ReservedNames() []string {
	ensureReservedCache()
	return append([]string(nil), reservedCache...)
}

// IsReserved reports whether name is a reserved built-in or stub.
func IsReserved(name string) bool {
	ensureReservedCache()
	for _, n := range reservedCache {
		if n == name {
			return true
		}
	}
	return false
}
