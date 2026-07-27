package config

import "strings"

// EnvSnapshot is a frozen environment map for one CLI invocation.
// Windows keys are normalized to uppercase; last duplicate wins.
type EnvSnapshot struct {
	goos string
	vars map[string]string
}

// NewEnvSnapshot parses KEY=VALUE pairs into a lookup map.
// Malformed entries without '=' use the whole string as key with an empty value.
func NewEnvSnapshot(env []string, goos string) EnvSnapshot {
	snap := EnvSnapshot{goos: goos, vars: make(map[string]string, len(env))}
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

// Lookup returns the value for key. On Windows lookup keys are case-insensitive.
func (e EnvSnapshot) Lookup(key string) (string, bool) {
	if e.vars == nil {
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
	if e.vars == nil {
		return EnvSnapshot{goos: e.goos}
	}
	out := make(map[string]string, len(e.vars))
	for k, v := range e.vars {
		out[k] = v
	}
	return EnvSnapshot{goos: e.goos, vars: out}
}

// Environ returns KEY=VALUE pairs for LoadOptions backward compatibility.
func (e EnvSnapshot) Environ() []string {
	if e.vars == nil {
		return nil
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

// populated reports whether the snapshot holds any variables.
func (e EnvSnapshot) populated() bool {
	return len(e.vars) > 0
}
