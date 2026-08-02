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
	"github.com/mewisme/mew/internal/fsx"
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
	Values string // pipe-joined allowed values; empty when free-form
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

// ownedKeys maps canonical keys to their type strings.
// Built from the registry at init time for lookup speed.
var ownedKeys = buildOwnedKeys()

func buildOwnedKeys() map[string]string {
	m := make(map[string]string, len(keyRegistry))
	for k, s := range keyRegistry {
		m[k] = string(s.Type)
	}
	return m
}

// allowedValuesByKey lists fixed enums for `m config list` VALUES.
// Built from the registry at init time.
var allowedValuesByKey = buildAllowedValues()

func buildAllowedValues() map[string]string {
	m := make(map[string]string, len(keyRegistry))
	for k, s := range keyRegistry {
		if s.Type == TypeEnum && len(s.Enum) > 0 {
			m[k] = strings.Join(s.Enum, "|")
		}
	}
	return m
}

// AllowedValues returns the pipe-joined settable values for key, or "" when free-form.
func AllowedValues(key string) string {
	if v, ok := allowedValuesByKey[key]; ok {
		return v
	}
	if ownedKeys[key] == "bool" {
		return "true|false"
	}
	return ""
}

// OwnedKeys returns the sorted list of owned config keys (canonical only).
func OwnedKeys() []string {
	keys := make([]string, 0, len(ownedKeys))
	for k := range ownedKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// defaults returns the canonical default map.
func defaults() map[string]any {
	return CanonicalDefaults()
}

// GlobalConfigPath resolves the user config.jsonc path.
// intentional: ambient env for m config writes outside app.New snapshot.
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

	_ = opts.IdentityMew

	mergeEnv(eff, snap)

	for k, v := range opts.CLI {
		canon := k
		if c := CanonicalKey(k); c != "" {
			canon = c
		}
		if err := validateKeyValue(canon, v); err != nil {
			return nil, err
		}
		eff.Values[canon] = Value{Raw: v, Source: SourceCLI, Path: "cli"}
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
		canon, isLegacy, known := resolveKey(k)
		if !known {
			if err := validateUnknownKey(k, v); err != nil {
				return apperr.Wrap(apperr.Config, "config.load", path+":"+k, err)
			}
			eff.Values[k] = Value{Raw: v, Source: src, Path: path}
			continue
		}
		if isLegacy {
			if existing, ok := eff.Values[canon]; ok && existing.Source == src {
				return apperr.New(apperr.Config, "config.load", path,
					fmt.Sprintf("conflicting keys %q and %q; remove one", k, canon))
			}
		}
		if err := validateKeyValue(canon, v); err != nil {
			return apperr.Wrap(apperr.Config, "config.load", path+":"+k, err)
		}
		eff.Values[canon] = Value{Raw: v, Source: src, Path: path}
	}
	return nil
}

func validateUnknownKey(key string, v any) error {
	_ = v
	if strings.HasPrefix(key, "registries.") {
		return nil
	}
	if strings.HasPrefix(key, "install.") || strings.HasPrefix(key, "registry.") ||
		strings.HasPrefix(key, "cache.") || strings.HasPrefix(key, "store.") ||
		strings.HasPrefix(key, "network.") || strings.HasPrefix(key, "resolve.") ||
		strings.HasPrefix(key, "runner.") || strings.HasPrefix(key, "lifecycle.") ||
		strings.HasPrefix(key, "link.") || strings.HasPrefix(key, "transaction.") ||
		strings.HasPrefix(key, "workspaces.") || strings.HasPrefix(key, "provenance.") ||
		strings.HasPrefix(key, "ui.") || strings.HasPrefix(key, "log.") ||
		strings.HasPrefix(key, "mx.") {
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

// envVarByKey maps canonical config keys to env var names.
var envVarByKey = map[string]string{
	"cache.dir":                        "MEW_CACHE_DIR",
	"store.dir":                        "MEW_STORE_DIR",
	"offline":                          "MEW_OFFLINE",
	"prefer_offline":                   "MEW_PREFER_OFFLINE",
	"resolve.auto_install_peers":       "MEW_RESOLVE_AUTO_INSTALL_PEERS",
	"resolve.strict_peer_dependencies": "MEW_RESOLVE_STRICT_PEER_DEPS",
	"resolve.reject_deprecated":        "MEW_RESOLVE_REJECT_DEPRECATED",
	"registry":                         "MEW_REGISTRY",
	"registry.auth_token_env":          "MEW_REGISTRY_AUTH_TOKEN_ENV",
	"lifecycle.enabled":                "MEW_EXPERIMENTAL_LIFECYCLE",
	"lifecycle.script_timeout":         "MEW_LIFECYCLE_SCRIPT_TIMEOUT",
	"workspaces.enabled":               "MEW_EXPERIMENTAL_WORKSPACES",
	"runner.direct_scripts.enabled":    "MEW_EXPERIMENTAL_DIRECT_SCRIPTS",
	"provenance.trusted_public_key":    "MEW_PROVENANCE_TRUSTED_PUBLIC_KEY",
	"runner.mx.cache.dir":              "MEW_MX_CACHE_DIR",
	"link.use_global_store":            "MEW_EXPERIMENTAL_GLOBAL_STORE",
	"ui.pager":                         "MEW_PAGER",
	"log.level":                        "MEW_LOG_LEVEL",
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
	set("cache.dir", identity)
	set("store.dir", identity)
	set("offline", parseBool)
	set("prefer_offline", parseBool)
	set("resolve.auto_install_peers", parseBool)
	set("resolve.strict_peer_dependencies", parseBool)
	set("resolve.reject_deprecated", parseBool)
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
		if spec := KeySpec(key); spec != nil && spec.Type == TypeEnum && len(spec.Enum) > 0 {
			if s == "" {
				return nil
			}
			for _, allowed := range spec.Enum {
				if s == allowed {
					return nil
				}
			}
			return fmt.Errorf("%s: want %s", key, strings.Join(spec.Enum, "|"))
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
	case "duration":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s: want duration string", key)
		}
		if s != "" {
			if _, err := ParseDuration(s); err != nil {
				return fmt.Errorf("%s: invalid duration %q: %w", key, s, err)
			}
		}
	case "enum":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s: want string (enum)", key)
		}
		if spec := KeySpec(key); spec != nil {
			if s == "" {
				return nil
			}
			for _, allowed := range spec.Enum {
				if s == allowed {
					return nil
				}
			}
			return fmt.Errorf("%s: want %s", key, strings.Join(spec.Enum, "|"))
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
	for _, r := range s {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) && r != '_' {
			return true
		}
	}
	return false
}

// Get returns one effective value. Accepts canonical and recognized legacy keys.
func Get(eff *Effective, key string) (Value, error) {
	if eff == nil {
		return Value{}, apperr.New(apperr.Config, "config.get", key, "nil config")
	}
	if v, ok := eff.Values[key]; ok {
		return v, nil
	}
	if canon := CanonicalKey(key); canon != "" {
		if v, ok := eff.Values[canon]; ok {
			return v, nil
		}
		return Value{}, apperr.New(apperr.Config, "config.get", key, "unknown key")
	}
	return Value{}, apperr.New(apperr.Config, "config.get", key, "unknown key")
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
			Values: AllowedValues(k),
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

// SetFile writes a single key into a JSONC file. Legacy keys are resolved to canonical form.
func SetFile(path, key string, raw any) error {
	canon, _, known := resolveKey(key)
	if !known {
		canon = key
	}
	if err := validateKeyValue(canon, normalizeForWrite(raw)); err != nil {
		return apperr.Wrap(apperr.Config, "config.set", key, err)
	}
	raw = normalizeForWrite(raw)
	return mutateFile(path, "config.set", func(existing map[string]any) (bool, error) {
		setNested(existing, canon, raw)
		return true, nil
	})
}

// UnsetFile removes a dotted key from a JSONC file. Legacy keys are resolved first.
func UnsetFile(path, key string) error {
	canon, _, known := resolveKey(key)
	if !known {
		canon = key
	}
	if _, err := validateKeyOwned(canon); err != nil {
		return apperr.Wrap(apperr.Config, "config.unset", key, err)
	}
	return mutateFile(path, "config.unset", func(existing map[string]any) (bool, error) {
		return unsetNested(existing, canon), nil
	})
}

// MigrateFile reads a config file and rewrites it with canonical keys.
// Returns the count of migrated keys.
func MigrateFile(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, apperr.Wrap(apperr.IO, "config.migrate", path, err)
	}
	if HasJSONCComments(b) {
		return 0, apperr.New(apperr.Config, "config.migrate", path, "file contains comments; edit manually or remove comments before migrating")
	}
	parsed, err := ParseJSONC(b)
	if err != nil {
		return 0, apperr.Wrap(apperr.Config, "config.migrate", path, err)
	}
	m, ok := parsed.(map[string]any)
	if !ok {
		return 0, apperr.New(apperr.Config, "config.migrate", path, "root must be object")
	}
	count := migrateMap(m)
	if count == 0 {
		return 0, nil
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return 0, apperr.Wrap(apperr.Internal, "config.migrate", path, err)
	}
	out = append(out, '\n')
	if err := fsx.PublishFileDurable(path, out, 0o644); err != nil {
		return 0, apperr.Wrap(apperr.IO, "config.migrate", path, err)
	}
	return count, nil
}

// CheckMigration reports which legacy keys exist and their canonical replacements.
func CheckMigration(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "config.migrate", path, err)
	}
	parsed, err := ParseJSONC(b)
	if err != nil {
		return nil, apperr.Wrap(apperr.Config, "config.migrate", path, err)
	}
	m, ok := parsed.(map[string]any)
	if !ok {
		return nil, nil
	}
	needed := map[string]string{}
	for k := range flattenLegacy(m, "") {
		if canon := CanonicalKey(k); canon != "" && canon != k {
			needed[k] = canon
		}
	}
	if len(needed) == 0 {
		return nil, nil
	}
	return needed, nil
}

func flattenLegacy(v any, prefix string) map[string]any {
	out := map[string]any{}
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if cm, ok := child.(map[string]any); ok {
				for ck, cv := range flattenLegacy(cm, key) {
					out[ck] = cv
				}
			} else {
				out[key] = child
			}
		}
	}
	return out
}

func migrateMap(m map[string]any) int {
	count := 0
	for k, v := range m {
		if cm, ok := v.(map[string]any); ok {
			count += migrateMap(cm)
			if len(cm) == 0 {
				delete(m, k)
			}
			continue
		}
		canon := CanonicalKey(k)
		if canon == "" || canon == k {
			continue
		}
		if _, exists := m[canon]; exists {
			continue
		}
		m[canon] = v
		delete(m, k)
		count++
	}
	return count
}

func mutateFile(path, op string, mutate func(map[string]any) (changed bool, err error)) error {
	var existing map[string]any
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, op, path, err)
	}
	if err == nil {
		if HasJSONCComments(b) {
			return apperr.New(apperr.Config, op, path, "file contains comments; edit manually or remove comments before mutating")
		}
		parsed, err := ParseJSONC(b)
		if err != nil {
			return apperr.Wrap(apperr.Config, op, path, err)
		}
		m, ok := parsed.(map[string]any)
		if !ok {
			return apperr.New(apperr.Config, op, path, "root must be object")
		}
		existing = m
	} else {
		existing = map[string]any{}
	}
	changed, err := mutate(existing)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, op, path, err)
	}
	out = append(out, '\n')
	if err := fsx.PublishFileDurable(path, out, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, op, path, err)
	}
	return nil
}

func validateKeyOwned(key string) (string, error) {
	kind, ok := ownedKeys[key]
	if !ok && strings.HasPrefix(key, "registries.") {
		return "string", nil
	}
	if !ok {
		return "", fmt.Errorf("unknown key %q", key)
	}
	return kind, nil
}

func normalizeForWrite(v any) any {
	switch t := v.(type) {
	case string:
		if b, err := parseBool(t); err == nil {
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
// Accepts canonical keys; legacy keys are resolved first.
func ParseValue(key, s string) (any, error) {
	canon, _, known := resolveKey(key)
	if !known && strings.HasPrefix(key, "registries.") {
		canon = key
		known = true
	}
	if !known {
		return nil, fmt.Errorf("unknown key %q", key)
	}
	kind := ownedKeys[canon]
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

func unsetNested(m map[string]any, dotted string) bool {
	return unsetNestedParts(m, strings.Split(dotted, "."))
}

func unsetNestedParts(m map[string]any, parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	if len(parts) == 1 {
		if _, ok := m[parts[0]]; !ok {
			return false
		}
		delete(m, parts[0])
		return true
	}
	next, ok := m[parts[0]].(map[string]any)
	if !ok {
		return false
	}
	changed := unsetNestedParts(next, parts[1:])
	if changed && len(next) == 0 {
		delete(m, parts[0])
	}
	return changed
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
