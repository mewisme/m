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
//
// Values holds the winning value per key. Layers retains what each source
// declared on its own, so a value shadowed by a higher layer is still
// recoverable through GetAtScope, ListAtScope, and Explain.
type Effective struct {
	Values map[string]Value
	Layers []Layer
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
// Delegates to GlobalConfigPathFromEnv so both callers share one policy.
func GlobalConfigPath() string {
	return GlobalConfigPathFromEnv(NewEnvSnapshot(os.Environ(), runtime.GOOS))
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

	if err := mergeEnv(eff, snap); err != nil {
		return nil, err
	}

	cliLayer := newLayer(SourceCLI, "cli")
	cliKeys := make([]string, 0, len(opts.CLI))
	for k := range opts.CLI {
		cliKeys = append(cliKeys, k)
	}
	sort.Strings(cliKeys)
	for _, k := range cliKeys {
		v := opts.CLI[k]
		canon := k
		if c := CanonicalKey(k); c != "" {
			canon = c
		}
		if err := validateKeyValue(canon, v); err != nil {
			return nil, err
		}
		if spec := KeySpec(canon); spec != nil {
			if err := validateRange(spec, v); err != nil {
				return nil, apperr.Wrap(apperr.Config, "config.cli", canon, err)
			}
		}
		eff.Values[canon] = Value{Raw: v, Source: SourceCLI, Path: "cli"}
		cliLayer.Values[canon] = v
	}
	if len(cliLayer.Values) > 0 {
		eff.Layers = append(eff.Layers, *cliLayer)
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
	// One validator backs both loading and `m config validate`, so a file that
	// the validate command calls invalid can never load successfully.
	//
	// No Scope is passed: the per-key writable-scope rule governs where `m config
	// set` may write a key, not whether an existing file may be read. A project
	// file that already carries a user-only key stays loadable — refusing it here
	// would break working installs on a rule about writes. `m config validate
	// --scope project` still reports it, which is where the user asks about
	// placement. Strict is off too: a legacy spelling still works and only warns.
	if verr := ValidateDocument(b, path, ValidateOptions{}).Err(); verr != nil {
		return verr
	}
	raw, err := ParseJSONC(b)
	if err != nil {
		return apperr.Wrap(apperr.Config, "config.load", path, err)
	}
	flat, err := flatten(raw)
	if err != nil {
		return apperr.Wrap(apperr.Config, "config.load", path, err)
	}
	layer := newLayer(src, path)
	// Sort so a legacy/canonical collision inside one file reports the same
	// way on every run rather than following map iteration order.
	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := flat[k]
		canon, _, known := resolveKey(k)
		if !known {
			// Validation already accepted this key, so it is a legal free-form
			// entry and is retained under the spelling the file used.
			eff.Values[k] = Value{Raw: v, Source: src, Path: path}
			layer.Values[k] = v
			continue
		}
		eff.Values[canon] = Value{Raw: v, Source: src, Path: path}
		layer.Values[canon] = v
	}
	if len(layer.Values) > 0 {
		eff.Layers = append(eff.Layers, *layer)
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

// EnvVarKeys returns the canonical keys settable through the environment.
func EnvVarKeys() []string {
	out := make([]string, 0, len(envVarByKey))
	for k := range envVarByKey {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func mergeEnv(eff *Effective, snap EnvSnapshot) error {
	// Iterate the mapping itself so a key can never be declared here and then
	// forgotten in an apply list, and take the type from the key registry so
	// coercion cannot disagree with the schema.
	keys := make([]string, 0, len(envVarByKey))
	for key := range envVarByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	layer := newLayer(SourceEnv, "env")
	for _, key := range keys {
		envKey := envVarByKey[key]
		v, ok := snap.Lookup(envKey)
		if !ok || v == "" {
			continue
		}
		raw, err := coerceEnvValue(key, v)
		if err != nil {
			// Fail closed: a malformed environment value is a configuration
			// error, not something to silently ignore.
			return apperr.Wrap(apperr.Config, "config.env", envKey, err)
		}
		if err := validateKeyValue(key, raw); err != nil {
			return apperr.Wrap(apperr.Config, "config.env", envKey, err)
		}
		// Ranges are part of the schema, so an environment value must clear the
		// same Minimum/Maximum a file value does.
		if spec := KeySpec(key); spec != nil {
			if err := validateRange(spec, raw); err != nil {
				return apperr.Wrap(apperr.Config, "config.env", envKey, err)
			}
		}
		eff.Values[key] = Value{Raw: raw, Source: SourceEnv, Path: envKey}
		layer.Values[key] = raw
	}
	if len(layer.Values) > 0 {
		eff.Layers = append(eff.Layers, *layer)
	}
	return nil
}

// coerceEnvValue converts an environment string into the type the key registry
// declares for key.
func coerceEnvValue(key, s string) (any, error) {
	switch ownedKeys[key] {
	case "bool":
		return parseBool(s)
	case "int":
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("invalid int %q", s)
		}
		return n, nil
	default:
		// string, duration, and enum all arrive as strings; validateKeyValue
		// enforces their content.
		return s, nil
	}
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
	layer := newLayer(src, path)
	for k, v := range m {
		eff.Values[k] = Value{Raw: v, Source: src, Path: path}
		layer.Values[k] = v
	}
	if len(layer.Values) > 0 {
		eff.Layers = append(eff.Layers, *layer)
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
			if err := validateEnvVarName(s); err != nil {
				return fmt.Errorf("%s: %w", key, err)
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

// validateEnvVarName enforces that a value naming an environment variable is
// actually a variable name and not the secret itself. POSIX names are
// [A-Z_][A-Z0-9_]*; a pasted token almost always carries lowercase letters,
// dots, slashes, or spaces, so the shape check catches it before it lands in
// a config file in plaintext.
//
// Length is deliberately not a signal here: legitimate names such as
// MEW_PROVENANCE_TRUSTED_PUBLIC_KEY are long.
func validateEnvVarName(s string) error {
	if s == "" {
		return nil
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return fmt.Errorf("environment variable name cannot start with a digit")
			}
		default:
			return fmt.Errorf("store the environment variable name (A-Z, 0-9, underscore), not the secret itself")
		}
	}
	return nil
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
	// A value that would fail validation on load must not be written in the
	// first place, so `config set` enforces schema ranges too.
	if spec := KeySpec(canon); spec != nil {
		if err := validateRange(spec, normalizeForWrite(raw)); err != nil {
			return apperr.Wrap(apperr.Config, "config.set", key, err)
		}
	}
	raw = normalizeForWrite(raw)
	return spliceFile(path, "config.set", func(src []byte) ([]byte, bool, error) {
		out, err := setJSONCPath(src, canon, raw)
		if err != nil {
			return nil, false, err
		}
		return out, true, nil
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
	return spliceFile(path, "config.unset", func(src []byte) ([]byte, bool, error) {
		return unsetJSONCPath(src, canon)
	})
}

// MigrateFile reads a config file and rewrites it with canonical keys,
// preserving comments. Returns the count of migrated keys.
func MigrateFile(path string) (int, error) {
	plan, err := PlanMigration(path)
	if err != nil {
		return 0, err
	}
	return plan.Apply()
}

// CheckMigration reports which legacy keys exist and their canonical replacements.
// The returned map is keyed by legacy key for lookup; use PlanMigration directly
// when ordered steps are needed for output.
func CheckMigration(path string) (map[string]string, error) {
	plan, err := PlanMigration(path)
	if err != nil {
		return nil, err
	}
	if plan.Empty() {
		return nil, nil
	}
	needed := make(map[string]string, len(plan.Steps))
	for _, s := range plan.Steps {
		needed[s.From] = s.To
	}
	return needed, nil
}

// flattenDotted flattens a nested map into dotted-path keys.
func flattenDotted(v any, prefix string) map[string]any {
	out := map[string]any{}
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if cm, ok := child.(map[string]any); ok {
				for ck, cv := range flattenDotted(cm, key) {
					out[ck] = cv
				}
			} else {
				out[key] = child
			}
		}
	}
	return out
}

// spliceFile applies a comment-preserving in-place edit to a JSONC config
// file. Comments, blank lines, and the author's indentation outside the
// edited span survive. Falls back to creating a fresh file when absent.
func spliceFile(path, op string, edit func(src []byte) (out []byte, changed bool, err error)) error {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, op, path, err)
	}
	if os.IsNotExist(err) {
		b = []byte("{}\n")
	}
	// Parse first: never write over a file we cannot understand.
	parsed, perr := ParseJSONC(b)
	if perr != nil {
		return apperr.Wrap(apperr.Config, op, path, perr)
	}
	if _, ok := parsed.(map[string]any); !ok {
		return apperr.New(apperr.Config, op, path, "root must be object")
	}
	out, changed, err := edit(b)
	if err != nil {
		return apperr.Wrap(apperr.Config, op, path, err)
	}
	// A no-op edit writes nothing, so unsetting an absent key never creates
	// the file.
	if !changed {
		return nil
	}
	// Verify the result still parses before publishing it.
	if _, err := ParseJSONC(out); err != nil {
		return apperr.Wrap(apperr.Internal, op, path, err)
	}
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

// ParseJSONC strips // and /* */ comments then unmarshals JSON.
// Duplicate object keys at any depth are rejected before decode, so no value
// is silently dropped by encoding/json's last-value-wins behavior.
func ParseJSONC(b []byte) (any, error) {
	stripped := blankJSONC(b)
	if err := detectDuplicates(stripped); err != nil {
		return nil, err
	}
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

// blankJSONC returns a copy of b with // and /* */ comment bytes replaced by
// spaces, preserving newlines and total length. Because length is preserved,
// byte offsets in the result map 1:1 onto b, so encoding/json decoder offsets
// taken over the result are valid spans in the original source. This is what
// makes comment-preserving in-place edits possible.
func blankJSONC(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	inStr := false
	esc := false
	for i := 0; i < len(out); i++ {
		c := out[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			switch c {
			case '\\':
				esc = true
			case '"':
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			continue
		}
		if c != '/' || i+1 >= len(out) {
			continue
		}
		switch out[i+1] {
		case '/':
			for ; i < len(out) && out[i] != '\n'; i++ {
				out[i] = ' '
			}
		case '*':
			out[i], out[i+1] = ' ', ' '
			i += 2
			for ; i < len(out); i++ {
				if out[i] == '*' && i+1 < len(out) && out[i+1] == '/' {
					out[i], out[i+1] = ' ', ' '
					i++
					break
				}
				if out[i] != '\n' {
					out[i] = ' '
				}
			}
		}
	}
	return out
}
