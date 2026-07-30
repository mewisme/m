// Package config loads layered global and project configuration with provenance.
package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mewisme/mew/internal/apperr"
)

// Source identifies which layer provided a value.
type Source string

const (
	SourceDefaults Source = "defaults"
	SourceGlobal   Source = "global"
	SourceProject  Source = "project"
	SourceEnv      Source = "env"
	SourceCLI      Source = "cli"
)

// Value is one effective setting with provenance.
type Value struct {
	Raw    any
	Source Source
	Path   string
}

// Effective is the merged configuration map (dotted keys).
type Effective struct {
	Values map[string]Value
	Env    EnvSnapshot
}

// Entry is a sorted list row for `m config list`.
type Entry struct {
	Key    string
	Value  string
	Env    string // env var that can set Key; empty when none
	Source Source
	Path   string
}

// LoadOptions controls discovery and overlays.
type LoadOptions struct {
	CWD         string
	GlobalPath  string // empty = default global path
	ProjectPath string // empty = <root>/m.jsonc
	ProjectRoot string // empty = CWD (caller may set after FindRoot)
	Env         []string
	EnvSnapshot EnvSnapshot
	CLI         map[string]any // already-parsed CLI overlays
	// RequireProjectConfig/RequireGlobalConfig make explicit --config paths mandatory.
	RequireProjectConfig bool
	RequireGlobalConfig  bool
	// IdentityMew when true skips branded PM config authority (always true for Load today).
	IdentityMew bool
}

var ownedKeys = map[string]string{
	"registry":                       "string",
	"install.linker":                 "string",
	"offline":                        "bool",
	"prefer-offline":                 "bool",
	"cache.dir":                      "string",
	"store.dir":                      "string",
	"link.use_global_store":          "bool",
	"resolve.autoInstallPeers":       "bool",
	"resolve.strictPeerDependencies": "bool",
	"resolve.rejectDeprecated":       "bool",
	"resolve.minimumReleaseAge":      "int",
	"network.timeout_ms":             "int",
	"network.proxy":                  "string",
	"network.ca_file":                "string",
	"registry.auth_token_env":        "string",
	"transaction.snapshot_retention": "int",
	"lifecycle.enabled":              "bool",
	"lifecycle.ignore_scripts":       "bool",
	"lifecycle.script_trust":         "string",
	"lifecycle.script_timeout":       "string",
	"workspaces.enabled":             "bool",
	"runner.direct_scripts.enabled":  "bool",
	"runner.mx.cache.retention_days": "int",
	"runner.mx.cache.dir":            "string",
	"provenance.trusted_public_key":  "string",
	"ui.output":                      "string",
	"ui.color":                       "string",
	"ui.progress":                    "string",
	"ui.unicode":                     "string",
	"ui.interactive":                 "string",
	"ui.accessible":                  "bool",
	"ui.summary":                     "bool",
	"ui.theme":                       "string",
	"ui.pager":                       "string",
	"log.level":                      "string",
}

// OwnedKeys returns the sorted list of owned config keys.
func OwnedKeys() []string {
	keys := make([]string, 0, len(ownedKeys))
	for k := range ownedKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func defaults() map[string]any {
	return map[string]any{
		"registry":                       "https://registry.npmjs.org",
		"install.linker":                 "auto",
		"offline":                        false,
		"prefer-offline":                 false,
		"cache.dir":                      "",
		"store.dir":                      "",
		"link.use_global_store":          false,
		"resolve.autoInstallPeers":       false,
		"resolve.strictPeerDependencies": true,
		"resolve.rejectDeprecated":       false,
		"resolve.minimumReleaseAge":      0,
		"network.timeout_ms":             60000,
		"network.proxy":                  "",
		"network.ca_file":                "",
		"registry.auth_token_env":        "",
		"transaction.snapshot_retention": 10,
		"lifecycle.enabled":              false,
		"lifecycle.ignore_scripts":       false,
		"lifecycle.script_trust":         "deny",
		"lifecycle.script_timeout":       "10m",
		"workspaces.enabled":             false,
		"runner.mx.cache.retention_days": 7,
		"runner.mx.cache.dir":            "",
		"ui.output":                      "auto",
		"ui.color":                       "auto",
		"ui.progress":                    "auto",
		"ui.unicode":                     "auto",
		"ui.interactive":                 "auto",
		"ui.accessible":                  false,
		"ui.summary":                     true,
		"ui.theme":                       "",
		"ui.pager":                       "",
		"log.level":                      "error",
	}
}

// GlobalConfigPath resolves the user config.jsonc path.
// intentional: ambient env for m config set --global outside app.New snapshot.
func GlobalConfigPath() string {
	if d := os.Getenv("MEW_CONFIG_DIR"); d != "" {
		return filepath.Join(d, "config.jsonc")
	}
	if home := os.Getenv("MEW_HOME"); home != "" {
		return filepath.Join(home, "config", "config.jsonc")
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("AppData")
		if base == "" {
			base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(base, "mew", "config.jsonc")
	}
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(userHome(), ".config")
	}
	return filepath.Join(cfg, "mew", "config.jsonc")
}

func userHome() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

// Load merges all layers into an Effective config.
func Load(ctx context.Context, opts LoadOptions) (*Effective, error) {
	_ = ctx
	if opts.CWD == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "config.load", "cwd", err)
		}
		opts.CWD = wd
	}
	root := opts.ProjectRoot
	if root == "" {
		root = opts.CWD
	}
	snap := opts.EnvSnapshot
	if !snap.Initialized() {
		env := opts.Env
		if env == nil {
			// intentional: unit-test compat when Load is called without a snapshot.
			env = os.Environ()
		}
		snap = NewEnvSnapshot(env, runtime.GOOS)
	}
	eff := &Effective{Values: map[string]Value{}, Env: snap}

	putMap(eff, defaults(), SourceDefaults, "defaults")

	gpath := opts.GlobalPath
	if gpath == "" {
		gpath = GlobalConfigPath()
	}
	if err := mergeFile(eff, gpath, SourceGlobal, opts.RequireGlobalConfig); err != nil {
		return nil, err
	}

	ppath := opts.ProjectPath
	if ppath == "" {
		ppath = filepath.Join(root, "m.jsonc")
	}
	if err := mergeFile(eff, ppath, SourceProject, opts.RequireProjectConfig); err != nil {
		return nil, err
	}

	// Mew identity: do not read branded PM config as authority.
	_ = opts.IdentityMew

	mergeEnv(eff, snap)

	for k, v := range opts.CLI {
		if err := validateKeyValue(k, v); err != nil {
			return nil, err
		}
		eff.Values[k] = Value{Raw: v, Source: SourceCLI, Path: "cli"}
	}
	return eff, nil
}

func mergeFile(eff *Effective, path string, src Source, required bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if required {
				return apperr.New(apperr.Config, "config.load", path, "explicit config file missing: "+path)
			}
			return nil
		}
		return apperr.Wrap(apperr.IO, "config.load", path, err)
	}
	raw, err := ParseJSONC(b)
	if err != nil {
		return apperr.Wrap(apperr.Config, "config.load", path, err)
	}
	flat, err := flatten(raw)
	if err != nil {
		return apperr.Wrap(apperr.Config, "config.load", path, err)
	}
	for k, v := range flat {
		if err := validateKeyValue(k, v); err != nil {
			return apperr.Wrap(apperr.Config, "config.load", path+":"+k, err)
		}
		eff.Values[k] = Value{Raw: v, Source: src, Path: path}
	}
	return nil
}

// envVarByKey is the authoritative config-key → env-var map for mergeEnv and
// `m config list`. Only names that real resolution reads belong here.
var envVarByKey = map[string]string{
	"cache.dir":                      "MEW_CACHE_DIR",
	"store.dir":                      "MEW_STORE_DIR",
	"offline":                        "MEW_OFFLINE",
	"prefer-offline":                 "MEW_PREFER_OFFLINE",
	"resolve.autoInstallPeers":       "MEW_RESOLVE_AUTO_INSTALL_PEERS",
	"resolve.strictPeerDependencies": "MEW_RESOLVE_STRICT_PEER_DEPS",
	"resolve.rejectDeprecated":       "MEW_RESOLVE_REJECT_DEPRECATED",
	"registry":                       "MEW_REGISTRY",
	"registry.auth_token_env":        "MEW_REGISTRY_AUTH_TOKEN_ENV",
	"lifecycle.enabled":              "MEW_EXPERIMENTAL_LIFECYCLE",
	"lifecycle.script_timeout":       "MEW_LIFECYCLE_SCRIPT_TIMEOUT",
	"workspaces.enabled":             "MEW_EXPERIMENTAL_WORKSPACES",
	"runner.direct_scripts.enabled":  "MEW_EXPERIMENTAL_DIRECT_SCRIPTS",
	"provenance.trusted_public_key":  "MEW_PROVENANCE_TRUSTED_PUBLIC_KEY",
	"runner.mx.cache.dir":            "MEW_MX_CACHE_DIR",
	"link.use_global_store":          "MEW_EXPERIMENTAL_GLOBAL_STORE",
	"ui.output":                      "MEW_OUTPUT",
	"ui.color":                       "MEW_COLOR",
	"ui.progress":                    "MEW_PROGRESS",
	"ui.unicode":                     "MEW_UNICODE",
	"ui.interactive":                 "MEW_INTERACTIVE",
	"ui.accessible":                  "MEW_ACCESSIBLE",
	"ui.pager":                       "MEW_PAGER",
	"log.level":                      "MEW_LOG_LEVEL",
}

// EnvVar returns the environment variable that can set key, or "" when none.
func EnvVar(key string) string {
	return envVarByKey[key]
}

func mergeEnv(eff *Effective, snap EnvSnapshot) {
	set := func(key string, coerce func(string) (any, error)) {
		envKey := envVarByKey[key]
		if envKey == "" {
			return
		}
		v, ok := snap.Lookup(envKey)
		if !ok || v == "" {
			return
		}
		raw, err := coerce(v)
		if err != nil {
			return
		}
		eff.Values[key] = Value{Raw: raw, Source: SourceEnv, Path: envKey}
	}
	identity := func(s string) (any, error) { return s, nil }
	// Only keys that Load merges into Effective (presentation/path gates may
	// still list env names in envVarByKey for `m config list` without merging).
	set("cache.dir", identity)
	set("store.dir", identity)
	set("offline", parseBool)
	set("prefer-offline", parseBool)
	set("resolve.autoInstallPeers", parseBool)
	set("resolve.strictPeerDependencies", parseBool)
	set("resolve.rejectDeprecated", parseBool)
	set("registry", identity)
	set("registry.auth_token_env", identity)
	set("lifecycle.enabled", parseBool)
	set("lifecycle.script_timeout", identity)
	set("workspaces.enabled", parseBool)
	set("runner.direct_scripts.enabled", parseBool)
	set("provenance.trusted_public_key", identity)
}

func parseBool(s string) (any, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return nil, fmt.Errorf("invalid bool %q", s)
	}
}

func putMap(eff *Effective, m map[string]any, src Source, path string) {
	for k, v := range m {
		eff.Values[k] = Value{Raw: v, Source: src, Path: path}
	}
}

func flatten(v any) (map[string]any, error) {
	out := map[string]any{}
	var walk func(prefix string, x any) error
	walk = func(prefix string, x any) error {
		switch t := x.(type) {
		case map[string]any:
			for k, child := range t {
				key := k
				if prefix != "" {
					key = prefix + "." + k
				}
				if err := walk(key, child); err != nil {
					return err
				}
			}
		default:
			if prefix == "" {
				return fmt.Errorf("config root must be an object")
			}
			out[prefix] = t
		}
		return nil
	}
	if err := walk("", v); err != nil {
		return nil, err
	}
	return out, nil
}

func validateKeyValue(key string, v any) error {
	kind, ok := ownedKeys[key]
	if !ok {
		if strings.HasPrefix(key, "registries.") {
			kind = "string"
			ok = true
		}
	}
	if !ok {
		if strings.HasPrefix(key, "install.") || strings.HasPrefix(key, "registry.") ||
			strings.HasPrefix(key, "cache.") || strings.HasPrefix(key, "store.") ||
			strings.HasPrefix(key, "network.") || strings.HasPrefix(key, "registries.") {
			return fmt.Errorf("unknown config key %q", key)
		}
		return fmt.Errorf("unknown config key %q", key)
	}
	switch kind {
	case "string":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s: want string", key)
		}
		if key == "registry.auth_token_env" {
			if looksLikeSecret(s) {
				return fmt.Errorf("%s: store env var name only, not a secret", key)
			}
		}
		if key == "install.linker" {
			switch s {
			case "auto", "hoisted", "isolated", "":
			default:
				return fmt.Errorf("install.linker: want auto|hoisted|isolated")
			}
		}
		if key == "lifecycle.script_trust" {
			switch s {
			case "allow", "deny", "ask", "":
			default:
				return fmt.Errorf("lifecycle.script_trust: want allow|deny|ask")
			}
		}
	case "bool":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("%s: want bool", key)
		}
	case "int":
		switch n := v.(type) {
		case float64:
			if n != float64(int(n)) {
				return fmt.Errorf("%s: want int", key)
			}
		case int:
		case json.Number:
			_, err := n.Int64()
			return err
		default:
			return fmt.Errorf("%s: want int", key)
		}
	}
	return nil
}

func looksLikeSecret(s string) bool {
	if s == "" {
		return false
	}
	if strings.Contains(s, ".") || strings.Contains(s, " ") {
		return true
	}
	if len(s) >= 20 {
		return true
	}
	// Env var names are typically UPPER_SNAKE.
	for _, r := range s {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) && r != '_' {
			return true
		}
	}
	return false
}

// Get returns one effective value.
func Get(eff *Effective, key string) (Value, error) {
	if eff == nil {
		return Value{}, apperr.New(apperr.Config, "config.get", key, "nil config")
	}
	v, ok := eff.Values[key]
	if !ok {
		return Value{}, apperr.New(apperr.Config, "config.get", key, "unknown key")
	}
	return v, nil
}

// List returns sorted entries.
func List(eff *Effective) []Entry {
	if eff == nil {
		return nil
	}
	keys := make([]string, 0, len(eff.Values))
	for k := range eff.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Entry, 0, len(keys))
	for _, k := range keys {
		v := eff.Values[k]
		out = append(out, Entry{
			Key:    k,
			Value:  formatRaw(v.Raw),
			Env:    EnvVar(k),
			Source: v.Source,
			Path:   v.Path,
		})
	}
	return out
}

func formatRaw(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// SetFile writes a single key into a JSONC file (pure JSON rewrite).
func SetFile(path, key string, raw any) error {
	if err := validateKeyValue(key, normalizeForWrite(raw)); err != nil {
		return apperr.Wrap(apperr.Config, "config.set", key, err)
	}
	raw = normalizeForWrite(raw)
	var existing map[string]any
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "config.set", path, err)
	}
	if err == nil {
		if HasJSONCComments(b) {
			return apperr.New(apperr.Config, "config.set", path, "file contains comments; edit manually or remove comments before set")
		}
		parsed, err := ParseJSONC(b)
		if err != nil {
			return apperr.Wrap(apperr.Config, "config.set", path, err)
		}
		m, ok := parsed.(map[string]any)
		if !ok {
			return apperr.New(apperr.Config, "config.set", path, "root must be object")
		}
		existing = m
	} else {
		existing = map[string]any{}
	}
	setNested(existing, key, raw)
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "config.set", path, err)
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "config.set", path, err)
	}
	return os.WriteFile(path, out, 0o644)
}

func normalizeForWrite(v any) any {
	switch t := v.(type) {
	case string:
		if b, err := parseBool(t); err == nil {
			// keep string unless key expects bool — caller validates
			_ = b
		}
		if n, err := strconv.Atoi(t); err == nil {
			_ = n
		}
		return t
	default:
		return v
	}
}

// ParseValue coerces a CLI string into a typed value for key.
func ParseValue(key, s string) (any, error) {
	kind, ok := ownedKeys[key]
	if !ok && strings.HasPrefix(key, "registries.") {
		kind = "string"
		ok = true
	}
	if !ok {
		return nil, fmt.Errorf("unknown key %q", key)
	}
	switch kind {
	case "bool":
		return parseBool(s)
	case "int":
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		return n, nil
	default:
		return s, nil
	}
}

func setNested(m map[string]any, dotted string, v any) {
	parts := strings.Split(dotted, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = v
			return
		}
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
}

// ParseJSONC strips // and /* */ comments then unmarshals JSON.
func ParseJSONC(b []byte) (any, error) {
	stripped := stripJSONC(b)
	dec := json.NewDecoder(bytes.NewReader(stripped))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return normalizeNumbers(v), nil
}

func normalizeNumbers(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			t[k] = normalizeNumbers(child)
		}
		return t
	case []any:
		for i, child := range t {
			t[i] = normalizeNumbers(child)
		}
		return t
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i)
		}
		f, _ := t.Float64()
		return f
	default:
		return v
	}
}

// HasJSONCComments reports whether b likely contains // or /* outside strings.
func HasJSONCComments(b []byte) bool {
	inStr := false
	esc := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			continue
		}
		if c == '/' && i+1 < len(b) {
			if b[i+1] == '/' || b[i+1] == '*' {
				return true
			}
		}
	}
	return false
}

func stripJSONC(b []byte) []byte {
	var out bytes.Buffer
	inStr := false
	esc := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inStr {
			out.WriteByte(c)
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			out.WriteByte(c)
			continue
		}
		if c == '/' && i+1 < len(b) {
			if b[i+1] == '/' {
				i += 2
				for i < len(b) && b[i] != '\n' {
					i++
				}
				if i < len(b) {
					out.WriteByte('\n')
				}
				continue
			}
			if b[i+1] == '*' {
				i += 2
				for i+1 < len(b) && (b[i] != '*' || b[i+1] != '/') {
					i++
				}
				i++ // skip '/'
				continue
			}
		}
		out.WriteByte(c)
	}
	return out.Bytes()
}
