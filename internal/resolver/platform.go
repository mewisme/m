package resolver

import (
	"os"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/registry"
)

// Target is the current install platform (npm os/cpu/libc vocabulary).
type Target struct {
	OS   string
	CPU  string
	Libc string // glibc, musl, or empty when unknown / not applicable
}

// CurrentTarget returns the host platform in npm packument vocabulary.
func CurrentTarget() Target {
	return Target{
		OS:   normalizeOS(runtime.GOOS),
		CPU:  normalizeCPU(runtime.GOARCH),
		Libc: detectLibc(),
	}
}

// Matches reports whether meta's optional/platform constraints include target.
// Empty constraint lists match every platform.
// npm-compatible: positive entries require a match; !prefixed entries exclude.
func (t Target) Matches(meta registry.VersionMeta) bool {
	if len(meta.OS) > 0 && !platformListMatches(meta.OS, t.OS) {
		return false
	}
	if len(meta.CPU) > 0 && !platformListMatches(meta.CPU, t.CPU) {
		return false
	}
	if len(meta.Libc) > 0 && t.Libc != "" && !platformListMatches(meta.Libc, t.Libc) {
		return false
	}
	return true
}

func platformListMatches(list []string, want string) bool {
	var hasPositive bool
	for _, entry := range list {
		neg := strings.HasPrefix(entry, "!")
		val := entry
		if neg {
			val = strings.TrimPrefix(entry, "!")
		}
		if neg {
			if strings.EqualFold(val, want) {
				return false
			}
			continue
		}
		hasPositive = true
		if strings.EqualFold(val, want) {
			return true
		}
	}
	return !hasPositive
}

func normalizeOS(goos string) string {
	if goos == "windows" {
		return "win32"
	}
	return goos
}

func normalizeCPU(goarch string) string {
	switch goarch {
	case "amd64":
		return "x64"
	case "386":
		return "ia32"
	default:
		return goarch
	}
}

// ponytail: musl probe is a single-file existence check; upgrade to ELF interpreter parse if false negatives appear.
func detectLibc() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	for _, path := range []string{
		"/lib/libc.musl-x86_64.so.1",
		"/lib/libc.musl-aarch64.so.1",
		"/lib/aarch64-linux-musl/libc.so",
	} {
		if _, err := os.Stat(path); err == nil {
			return "musl"
		}
	}
	return "glibc"
}
