package config

import "strings"

// EnvSnapshot is a frozen environment map for one CLI invocation.
// Windows keys are normalized to uppercase; last duplicate wins.
type EnvSnapshot struct {
	initialized bool
	goos        string
	vars        map[string]string // non-nil when initialized (may be empty)
}

// NewEnvSnapshot parses KEY=VALUE pairs into a lookup map.
// Malformed entries without '=' use the whole string as key with an empty value.
// nil or empty env produces an initialized-empty snapshot (not ambient).
func NewEnvSnapshot(env []string, goos string) EnvSnapshot {
	snap := EnvSnapshot{initialized: true, goos: goos, vars: make(map[string]string, len(env))}
	for _, e := range env {
		key, val, ok := strings.Cut(e, "=")
		if !ok {
			val = ""
		}
		if goos == "windows" {
			key = strings.ToUpper(key)
		}
		snap.vars[key] = val
	}
	return snap
}

// Initialized reports whether the snapshot was explicitly built (including initialized-empty).
// Zero-value EnvSnapshot is uninitialized and may fall back to ambient env in unit tests.
func (e EnvSnapshot) Initialized() bool {
	return e.initialized
}

// Lookup returns the value for key. On Windows lookup keys are case-insensitive.
// Initialized-empty snapshots never consult ambient env.
func (e EnvSnapshot) Lookup(key string) (string, bool) {
	if !e.initialized || e.vars == nil {
		return "", false
	}
	k := key
	if e.goos == "windows" {
		k = strings.ToUpper(key)
	}
	v, ok := e.vars[k]
	return v, ok
}

// Clone returns a deep copy of the snapshot.
func (e EnvSnapshot) Clone() EnvSnapshot {
	if !e.initialized {
		return EnvSnapshot{goos: e.goos}
	}
	out := make(map[string]string, len(e.vars))
	for k, v := range e.vars {
		out[k] = v
	}
	return EnvSnapshot{initialized: true, goos: e.goos, vars: out}
}

// Environ returns KEY=VALUE pairs for LoadOptions backward compatibility.
// Initialized-empty returns a non-nil empty slice; uninitialized returns nil.
func (e EnvSnapshot) Environ() []string {
	if !e.initialized {
		return nil
	}
	if len(e.vars) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(e.vars))
	for k, v := range e.vars {
		out = append(out, k+"="+v)
	}
	return out
}

// GOOS returns the platform the snapshot was built for.
func (e EnvSnapshot) GOOS() string {
	return e.goos
}
