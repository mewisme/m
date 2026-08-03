package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
)

// These tests drive the production invocation path (runInvocation) so what they
// assert is what a real `m config ...` call does: bootstrap loads configuration
// once, and the config commands read the layers it retained.

// layeredFixture is a project plus user config with a known value at every layer.
type layeredFixture struct {
	proj    string
	cfgDir  string
	cfgFile string
}

// newLayeredFixture writes user and project config files under an isolated home.
// registry is used as the probe key because it is writable in both scopes and
// settable from the environment.
func newLayeredFixture(t *testing.T, userBody, projectBody string) layeredFixture {
	t.Helper()
	env := testkit.CleanEnv(t)
	clearHelpEnv(t)

	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if projectBody != "" {
		if err := os.WriteFile(filepath.Join(proj, "m.jsonc"), []byte(projectBody), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfgFile := filepath.Join(env.ConfigDir, "config.jsonc")
	if userBody != "" {
		if err := os.WriteFile(cfgFile, []byte(userBody), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return layeredFixture{proj: proj, cfgDir: env.ConfigDir, cfgFile: cfgFile}
}

// run executes argv through the production invocation path and returns stdout.
func (f layeredFixture) run(t *testing.T, argv ...string) (string, int) {
	t.Helper()
	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	full := append([]string{"--cwd", f.proj}, argv...)
	code := runInvocation(context.Background(), root, testBuildInfo(), full)
	return buf.String(), code
}

// runOK fails the test when the invocation did not succeed.
func (f layeredFixture) runOK(t *testing.T, argv ...string) string {
	t.Helper()
	out, code := f.run(t, argv...)
	if code != 0 {
		t.Fatalf("argv=%v exit=%d out=%s", argv, code, out)
	}
	return out
}

// ── §14 core scope matrix ───────────────────────────────────────────────────

// TestConfigGetScopeMatrix is the central proof of the layered model:
// user=A, project=B, environment=C, CLI=D, and each scope reports its own value
// while effective reports the winner.
func TestConfigGetScopeMatrix(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example"}`,
		`{"registry":"https://project.example"}`)
	// C: environment layer. D: CLI layer, via the --offline style overlay is not
	// available for registry, so the top layer here is env and the CLI rung is
	// asserted separately with a key the CLI can set.
	t.Setenv("MEW_REGISTRY", "https://env.example")

	if got := strings.TrimSpace(f.runOK(t, "config", "get", "registry", "--scope", "user")); got != "https://user.example" {
		t.Errorf("user scope = %q, want the user value", got)
	}
	if got := strings.TrimSpace(f.runOK(t, "config", "get", "registry", "--scope", "project")); got != "https://project.example" {
		t.Errorf("project scope = %q, want the project value", got)
	}
	if got := strings.TrimSpace(f.runOK(t, "config", "get", "registry", "--scope", "effective")); got != "https://env.example" {
		t.Errorf("effective scope = %q, want the env value", got)
	}
	// Default scope is user.
	if got := strings.TrimSpace(f.runOK(t, "config", "get", "registry")); got != "https://user.example" {
		t.Errorf("default scope = %q, want the user value", got)
	}
}

// TestConfigGetEffectiveHonorsCLIOverlay pins the top of the chain: a CLI
// overlay outranks user, project, and environment.
func TestConfigGetEffectiveHonorsCLIOverlay(t *testing.T) {
	f := newLayeredFixture(t, `{"offline":false}`, `{"offline":false}`)
	t.Setenv("MEW_OFFLINE", "0")

	if got := strings.TrimSpace(f.runOK(t, "--offline", "config", "get", "offline", "--scope", "effective")); got != "true" {
		t.Errorf("effective offline = %q, want true from the CLI overlay", got)
	}
	// The lower layers are unchanged by the overlay.
	if got := strings.TrimSpace(f.runOK(t, "--offline", "config", "get", "offline", "--scope", "user")); got != "false" {
		t.Errorf("user offline = %q, want the user value false", got)
	}
}

// TestConfigGetShadowedValuesStayReadable is the anti-regression for scope
// simulation: a value overridden by a higher layer must still be readable at
// its own scope.
func TestConfigGetShadowedValuesStayReadable(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example"}`,
		`{"registry":"https://project.example"}`)

	// User survives a project override.
	if got := strings.TrimSpace(f.runOK(t, "config", "get", "registry", "--scope", "user")); got != "https://user.example" {
		t.Errorf("user value lost to project override: %q", got)
	}

	// Project survives an environment override.
	t.Setenv("MEW_REGISTRY", "https://env.example")
	if got := strings.TrimSpace(f.runOK(t, "config", "get", "registry", "--scope", "project")); got != "https://project.example" {
		t.Errorf("project value lost to env override: %q", got)
	}
}

// TestConfigGetRawUnsetIsTypedError checks that a raw scope reports absence
// rather than borrowing a value from another layer or from the schema.
func TestConfigGetRawUnsetIsTypedError(t *testing.T) {
	f := newLayeredFixture(t, "", `{"registry":"https://project.example"}`)

	out, code := f.run(t, "config", "get", "registry", "--scope", "user")
	if code == 0 {
		t.Fatalf("unset user key must fail, got %q", out)
	}
	if !strings.Contains(out, string(apperr.Config)) {
		t.Errorf("want a typed config error in output, got %q", out)
	}
	// It must not have leaked the project value.
	if strings.Contains(out, "project.example") {
		t.Errorf("user scope leaked the project value: %q", out)
	}
}

// TestConfigGetNotSetPreservesErrNotSet keeps errors.Is working internally
// while the user-facing code stays ERR_M_CONFIG.
func TestConfigGetNotSetPreservesErrNotSet(t *testing.T) {
	err := newNotSetError("registry", configScopeUser)
	if !errors.Is(err, config.ErrNotSet) {
		t.Error("errors.Is(err, config.ErrNotSet) must hold")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Errorf("code=%s want %s", apperr.CodeOf(err), apperr.Config)
	}
	if !strings.Contains(err.Error(), "registry") || !strings.Contains(err.Error(), "user") {
		t.Errorf("message must name key and scope: %q", err.Error())
	}
}

// TestConfigGetDefaultOnlyKeyNotConfiguredInUserScope proves a schema default
// never counts as a user-scope value.
func TestConfigGetDefaultOnlyKeyNotConfiguredInUserScope(t *testing.T) {
	f := newLayeredFixture(t, "", "")

	if _, code := f.run(t, "config", "get", "install.linker", "--scope", "user"); code == 0 {
		t.Error("default-only key must not be reported as configured in user scope")
	}
	// The default is still the effective value.
	if got := strings.TrimSpace(f.runOK(t, "config", "get", "install.linker", "--scope", "effective")); got == "" {
		t.Error("effective scope must return the schema default")
	}
}

// ── §4 get --json ───────────────────────────────────────────────────────────

func TestConfigGetJSONScopeSemantics(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example"}`,
		`{"registry":"https://project.example"}`)
	t.Setenv("MEW_REGISTRY", "https://env.example")

	var doc map[string]any
	out := f.runOK(t, "--output", "json", "config", "get", "registry", "--scope", "user")
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	if doc["value"] != "https://user.example" {
		t.Errorf("value=%v want the user value", doc["value"])
	}
	if doc["effective_value"] != "https://env.example" {
		t.Errorf("effective_value=%v want the env value", doc["effective_value"])
	}
	if doc["scope"] != "user" || doc["source"] != "user" {
		t.Errorf("scope=%v source=%v want user/user", doc["scope"], doc["source"])
	}
	if doc["configured"] != true {
		t.Errorf("configured=%v want true", doc["configured"])
	}
	if doc["is_default"] != false {
		t.Errorf("is_default=%v want false", doc["is_default"])
	}
	if doc["type"] != "string" {
		t.Errorf("type=%v want string", doc["type"])
	}
	// Structured output carries no ANSI and no human header.
	if strings.Contains(out, "\x1b[") {
		t.Errorf("structured output contains ANSI: %q", out)
	}
}

func TestConfigGetJSONUnsetReportsConfiguredFalse(t *testing.T) {
	f := newLayeredFixture(t, "", `{"registry":"https://project.example"}`)

	out, code := f.run(t, "--output", "json", "config", "get", "registry", "--scope", "user")
	if code == 0 {
		t.Fatal("unset raw scope must still exit non-zero")
	}
	// The structured document precedes the typed error.
	line := strings.SplitN(strings.TrimSpace(out), "\n", 2)[0]
	var doc map[string]any
	if err := json.Unmarshal([]byte(line), &doc); err != nil {
		t.Fatalf("json: %v line=%s", err, line)
	}
	if doc["configured"] != false {
		t.Errorf("configured=%v want false", doc["configured"])
	}
	if _, ok := doc["value"]; ok && doc["value"] != nil {
		t.Errorf("unset raw value must be absent or null, got %v", doc["value"])
	}
	if doc["effective_value"] != "https://project.example" {
		t.Errorf("effective_value=%v want the project value", doc["effective_value"])
	}
}

// ── §5 and §6 list ──────────────────────────────────────────────────────────

func TestConfigListRawScopesListOnlyTheirOwnValues(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example"}`,
		`{"install":{"linker":"isolated"}}`)

	user := f.runOK(t, "config", "list")
	if !strings.Contains(user, "registry") {
		t.Errorf("user list missing its own key:\n%s", user)
	}
	if strings.Contains(user, "install.linker") {
		t.Errorf("user list leaked a project key:\n%s", user)
	}

	proj := f.runOK(t, "config", "list", "--scope", "project")
	if !strings.Contains(proj, "install.linker") {
		t.Errorf("project list missing its own key:\n%s", proj)
	}
	if strings.Contains(proj, "https://user.example") {
		t.Errorf("project list leaked the user value:\n%s", proj)
	}

	// Effective lists the merged configuration, which includes both plus defaults.
	effective := f.runOK(t, "config", "list", "--scope", "effective")
	for _, want := range []string{"registry", "install.linker"} {
		if !strings.Contains(effective, want) {
			t.Errorf("effective list missing %q:\n%s", want, effective)
		}
	}
}

func TestConfigListDefaultsAddsUnconfiguredRows(t *testing.T) {
	f := newLayeredFixture(t, `{"registry":"https://user.example"}`, "")

	plain := f.runOK(t, "config", "list")
	withDefaults := f.runOK(t, "config", "list", "--defaults")
	if len(withDefaults) <= len(plain) {
		t.Errorf("--defaults must add rows:\nplain=%s\ndefaults=%s", plain, withDefaults)
	}
	if !strings.Contains(withDefaults, "install.linker") {
		t.Errorf("--defaults missing a default-only key:\n%s", withDefaults)
	}

	// The default row is reported as not configured.
	out := f.runOK(t, "--output", "json", "config", "list", "--defaults")
	entries := decodeListEntries(t, out)
	found := false
	for _, e := range entries {
		if e["key"] == "install.linker" {
			found = true
			if e["configured"] != false {
				t.Errorf("default row configured=%v want false", e["configured"])
			}
			if e["is_default"] != true {
				t.Errorf("default row is_default=%v want true", e["is_default"])
			}
		}
		if e["key"] == "registry" && e["configured"] != true {
			t.Errorf("configured row configured=%v want true", e["configured"])
		}
	}
	if !found {
		t.Error("--defaults did not emit the default row in JSON")
	}
}

func TestConfigListChangedFiltersDefaultValues(t *testing.T) {
	// A user file that re-states the schema default plus one real change.
	f := newLayeredFixture(t, `{"install":{"linker":"hoisted"},"registry":"https://user.example"}`, "")

	spec := config.KeySpec("install.linker")
	if spec == nil || spec.Default != "hoisted" {
		t.Skipf("fixture assumes install.linker defaults to hoisted, got %v", spec)
	}

	out := f.runOK(t, "config", "list", "--changed")
	if strings.Contains(out, "install.linker") {
		t.Errorf("--changed must drop a value equal to its default:\n%s", out)
	}
	if !strings.Contains(out, "registry") {
		t.Errorf("--changed dropped a genuinely changed value:\n%s", out)
	}

	// JSON must filter identically.
	jsonOut := f.runOK(t, "--output", "json", "config", "list", "--changed")
	for _, e := range decodeListEntries(t, jsonOut) {
		if e["key"] == "install.linker" {
			t.Errorf("--changed leaked a default-valued key into JSON: %v", e)
		}
	}
}

func TestConfigListPrefixFiltersNamespace(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example","registry.auth_token_env":"TOKEN_VAR","offline":true}`, "")

	out := f.runOK(t, "config", "list", "--prefix", "registry")
	if !strings.Contains(out, "registry") {
		t.Errorf("prefix dropped the exact key:\n%s", out)
	}
	if !strings.Contains(out, "registry.auth_token_env") {
		t.Errorf("prefix dropped a child key:\n%s", out)
	}
	if strings.Contains(out, "offline") {
		t.Errorf("prefix leaked an unrelated key:\n%s", out)
	}

	// JSON honours the same filter.
	entries := decodeListEntries(t, f.runOK(t, "--output", "json", "config", "list", "--prefix", "registry"))
	if len(entries) == 0 {
		t.Fatal("prefix filter returned no JSON entries")
	}
	for _, e := range entries {
		key, _ := e["key"].(string)
		if key != "registry" && !strings.HasPrefix(key, "registry.") {
			t.Errorf("prefix leaked %q into JSON", key)
		}
	}
}

func TestConfigListShowOriginReportsSelectedScope(t *testing.T) {
	f := newLayeredFixture(t, `{"registry":"https://user.example"}`, "")

	out := f.runOK(t, "config", "list", "--show-origin")
	if !strings.Contains(out, "user") {
		t.Errorf("--show-origin must name the source:\n%s", out)
	}
	if !strings.Contains(out, "config.jsonc") {
		t.Errorf("--show-origin must name the file:\n%s", out)
	}

	// source stays available in JSON without --show-origin; path is the opt-in.
	entries := decodeListEntries(t, f.runOK(t, "--output", "json", "config", "list"))
	for _, e := range entries {
		if e["key"] != "registry" {
			continue
		}
		if e["source"] != "user" {
			t.Errorf("source=%v want user", e["source"])
		}
		if p, ok := e["path"]; ok && p != "" {
			t.Errorf("path must be opt-in, got %v", p)
		}
	}
}

func TestConfigListJSONIsDeterministicAndClean(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example","offline":true,"install":{"linker":"isolated"}}`, "")

	first := f.runOK(t, "--output", "json", "config", "list")
	second := f.runOK(t, "--output", "json", "config", "list")
	if first != second {
		t.Errorf("list JSON is not deterministic:\n%s\n%s", first, second)
	}
	if strings.Contains(first, "\x1b[") {
		t.Errorf("structured output contains ANSI: %q", first)
	}
	if strings.Contains(first, "configured,") {
		t.Errorf("structured output contains the human summary: %q", first)
	}
}

// decodeListEntries unwraps the entries array from list JSON output.
func decodeListEntries(t *testing.T, out string) []map[string]any {
	t.Helper()
	var doc struct {
		Scope   string           `json:"scope"`
		Entries []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	return doc.Entries
}

// ── §7 and §8 explain ───────────────────────────────────────────────────────

func TestConfigExplainShowsOrderedChain(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example"}`,
		`{"registry":"https://project.example"}`)
	t.Setenv("MEW_REGISTRY", "https://env.example")

	out := f.runOK(t, "config", "explain", "registry")
	chain := resolutionLayers(t, out)

	// Every configured layer appears, in precedence order.
	want := []string{"defaults", "user", "project", "env"}
	if len(chain) != len(want) {
		t.Fatalf("chain has %d rungs (%v), want %d:\n%s", len(chain), chain, len(want), out)
	}
	for i, layer := range want {
		if chain[i] != layer {
			t.Errorf("rung %d is %q, want %q:\n%s", i, chain[i], layer, out)
		}
	}
	// Exactly one winner, and it is the highest layer present.
	if n := strings.Count(out, "<- effective"); n != 1 {
		t.Errorf("%d layers marked effective, want exactly 1:\n%s", n, out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "<- effective") && !strings.HasPrefix(strings.TrimSpace(line), "env") {
			t.Errorf("effective marker on the wrong layer: %q", line)
		}
	}
}

// resolutionLayers returns the layer names listed under the Resolution heading,
// in the order they were rendered. Parsing the block rather than searching the
// whole output keeps a value that happens to contain a layer name from being
// mistaken for a rung.
func resolutionLayers(t *testing.T, out string) []string {
	t.Helper()
	var layers []string
	inBlock := false
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "Resolution" {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			break // blank line ends the block
		}
		layers = append(layers, fields[0])
	}
	return layers
}

func TestConfigExplainMarksOneWinnerWithDuplicateValues(t *testing.T) {
	// Two layers holding the same value must not both claim to be effective.
	f := newLayeredFixture(t,
		`{"registry":"https://same.example"}`,
		`{"registry":"https://same.example"}`)

	out := f.runOK(t, "config", "explain", "registry")
	if n := strings.Count(out, "<- effective"); n != 1 {
		t.Errorf("%d layers marked effective with duplicate values, want 1:\n%s", n, out)
	}

	doc := decodeExplain(t, f.runOK(t, "--output", "json", "config", "explain", "registry"))
	marked := 0
	for _, l := range doc.Layers {
		if l.Effective {
			marked++
			if l.Source != "project" {
				t.Errorf("effective layer is %q, want the highest (project)", l.Source)
			}
		}
	}
	if marked != 1 {
		t.Errorf("%d JSON layers marked effective, want 1", marked)
	}
}

func TestConfigExplainJSONCarriesOrderedLayers(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example"}`,
		`{"registry":"https://project.example"}`)

	doc := decodeExplain(t, f.runOK(t, "--output", "json", "config", "explain", "registry"))
	if doc.Key != "registry" {
		t.Errorf("key=%q", doc.Key)
	}
	if doc.EffectiveValue != "https://project.example" {
		t.Errorf("effective_value=%v want the project value", doc.EffectiveValue)
	}
	if doc.Type != "string" {
		t.Errorf("type=%q want string", doc.Type)
	}
	if len(doc.Scopes) == 0 {
		t.Error("scopes must be reported")
	}
	if doc.Description == "" {
		t.Error("description must be reported")
	}
	wantSources := []string{"defaults", "user", "project"}
	if len(doc.Layers) != len(wantSources) {
		t.Fatalf("got %d layers (%+v), want %d", len(doc.Layers), doc.Layers, len(wantSources))
	}
	for i, want := range wantSources {
		if doc.Layers[i].Source != want {
			t.Errorf("layer %d source=%q want %q", i, doc.Layers[i].Source, want)
		}
		if !doc.Layers[i].Configured {
			t.Errorf("layer %d must be marked configured", i)
		}
	}
	if doc.Layers[1].Value != "https://user.example" {
		t.Errorf("shadowed user layer value=%v", doc.Layers[1].Value)
	}
}

type explainDoc struct {
	Key            string   `json:"key"`
	Value          any      `json:"value"`
	EffectiveValue any      `json:"effective_value"`
	Source         string   `json:"source"`
	Type           string   `json:"type"`
	Default        any      `json:"default"`
	Scopes         []string `json:"scopes"`
	Description    string   `json:"description"`
	IsSecret       bool     `json:"is_secret"`
	Layers         []struct {
		Source     string `json:"source"`
		Value      any    `json:"value"`
		Path       string `json:"path"`
		Configured bool   `json:"configured"`
		Effective  bool   `json:"effective"`
	} `json:"layers"`
}

func decodeExplain(t *testing.T, out string) explainDoc {
	t.Helper()
	var doc explainDoc
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	return doc
}

// ── §9 set ──────────────────────────────────────────────────────────────────

// TestConfigSetReportsPreviousTargetScopeValue is the anti-regression for
// reporting the effective winner as "Previous": the user file's own value is
// what a user-scope write replaces, even when a project value outranks it.
func TestConfigSetReportsPreviousTargetScopeValue(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example"}`,
		`{"registry":"https://project.example"}`)

	out := f.runOK(t, "config", "set", "registry", "https://new.example")
	if !strings.Contains(out, "https://user.example") {
		t.Errorf("Previous must be the user value:\n%s", out)
	}
	if strings.Contains(out, "Previous") && strings.Contains(out, "project.example") {
		// The project value may legitimately appear as Effective; it must not be
		// the Previous row.
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "Previous") && strings.Contains(line, "project.example") {
				t.Errorf("Previous reported the project value: %q", line)
			}
		}
	}
	// The project layer still wins, and that is reported separately.
	if !strings.Contains(out, "Effective") {
		t.Errorf("a shadowed write must report the effective value:\n%s", out)
	}
}

func TestConfigSetReportsUnsetPrevious(t *testing.T) {
	f := newLayeredFixture(t, "", "")

	out := f.runOK(t, "config", "set", "registry", "https://new.example")
	if !strings.Contains(out, "(unset)") {
		t.Errorf("a first write must report no previous value:\n%s", out)
	}
	if !strings.Contains(out, "https://new.example") {
		t.Errorf("Current missing:\n%s", out)
	}
}

func TestConfigSetProjectScopeLeavesUserFileAlone(t *testing.T) {
	f := newLayeredFixture(t, `{"registry":"https://user.example"}`, `{}`)

	f.runOK(t, "config", "set", "registry", "https://project-new.example", "--scope", "project")

	user, err := os.ReadFile(f.cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(user), "user.example") {
		t.Errorf("user file changed by a project write:\n%s", user)
	}
	proj, err := os.ReadFile(filepath.Join(f.proj, "m.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proj), "project-new.example") {
		t.Errorf("project file not written:\n%s", proj)
	}
}

// ── §10 unset ───────────────────────────────────────────────────────────────

func TestConfigUnsetRevealsFallbackLayer(t *testing.T) {
	f := newLayeredFixture(t,
		`{"registry":"https://user.example"}`,
		`{"registry":"https://project.example"}`)

	out := f.runOK(t, "config", "unset", "registry", "--scope", "project")
	// With the project value gone, the user value becomes effective.
	if !strings.Contains(out, "https://user.example") {
		t.Errorf("unset must report the new fallback value:\n%s", out)
	}
	if !strings.Contains(out, "user") {
		t.Errorf("unset must name the fallback source:\n%s", out)
	}
	// Only the project file was edited.
	user, err := os.ReadFile(f.cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(user), "user.example") {
		t.Errorf("unset removed another layer's value:\n%s", user)
	}
}

func TestConfigUnsetAlreadyUnsetIsNoOp(t *testing.T) {
	f := newLayeredFixture(t, `{"offline":true}`, "")

	// registry is not in the user file; removing it changes nothing.
	before, err := os.ReadFile(f.cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, code := f.run(t, "config", "unset", "registry"); code != 0 {
		t.Error("unsetting an absent key must succeed")
	}
	after, err := os.ReadFile(f.cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("no-op unset rewrote the file:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestConfigUnsetEffectiveScopeRejected(t *testing.T) {
	f := newLayeredFixture(t, `{"registry":"https://user.example"}`, "")

	out, code := f.run(t, "config", "unset", "registry", "--scope", "effective")
	if code == 0 {
		t.Fatalf("effective scope must be read-only, got %q", out)
	}
	if !strings.Contains(out, string(apperr.Usage)) {
		t.Errorf("want a usage error:\n%s", out)
	}
}

// ── §11 schema-driven scope validation ──────────────────────────────────────

func TestConfigWriteScopeValidationIsSchemaDriven(t *testing.T) {
	f := newLayeredFixture(t, "", `{}`)

	// ui.theme is user-only in the schema.
	out, code := f.run(t, "config", "set", "ui.theme", "dark", "--scope", "project")
	if code == 0 {
		t.Fatalf("user-only key must reject a project write, got %q", out)
	}
	for _, want := range []string{"ui.theme", "user", "project"} {
		if !strings.Contains(out, want) {
			t.Errorf("error must name %q:\n%s", want, out)
		}
	}

	// The same key is accepted in user scope.
	if _, code := f.run(t, "config", "set", "ui.theme", "dark"); code != 0 {
		t.Error("user-only key must be writable in user scope")
	}
	// unset enforces the same policy.
	if _, code := f.run(t, "config", "unset", "ui.theme", "--scope", "project"); code == 0 {
		t.Error("unset must enforce the schema scope list too")
	}
}

func TestCheckWritableScopeMatchesSchema(t *testing.T) {
	// A user-only key rejects project and accepts user.
	if err := checkWritableScope("ui.theme", configScopeProject); err == nil {
		t.Error("ui.theme must reject project scope")
	}
	if err := checkWritableScope("ui.theme", configScopeUser); err != nil {
		t.Errorf("ui.theme must accept user scope: %v", err)
	}
	// A both-scope key accepts either.
	for _, s := range []configScope{configScopeUser, configScopeProject} {
		if err := checkWritableScope("registry", s); err != nil {
			t.Errorf("registry must accept %s: %v", s, err)
		}
	}
	// Dynamic registries.* keys have no spec and keep their any-scope policy.
	if err := checkWritableScope("registries.//npm.example/", configScopeProject); err != nil {
		t.Errorf("dynamic registries key policy changed: %v", err)
	}
}

// ── §14 secret redaction ────────────────────────────────────────────────────

// TestConfigSecretsRedactedEverywhere covers get, list, and explain in both
// human and structured output. registry.auth_token_env is declared secret.
func TestConfigSecretsRedactedEverywhere(t *testing.T) {
	const secret = "MY_SECRET_TOKEN_VAR"
	f := newLayeredFixture(t, `{"registry":{"auth_token_env":"`+secret+`"}}`, "")

	cases := []struct {
		name string
		argv []string
	}{
		{"get", []string{"config", "get", "registry.auth_token_env"}},
		{"get-verbose", []string{"config", "get", "registry.auth_token_env", "--verbose"}},
		{"get-json", []string{"--output", "json", "config", "get", "registry.auth_token_env"}},
		{"get-effective", []string{"config", "get", "registry.auth_token_env", "--scope", "effective"}},
		{"list", []string{"config", "list"}},
		{"list-origin", []string{"config", "list", "--show-origin"}},
		{"list-json", []string{"--output", "json", "config", "list"}},
		{"explain", []string{"config", "explain", "registry.auth_token_env"}},
		{"explain-json", []string{"--output", "json", "config", "explain", "registry.auth_token_env"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := f.runOK(t, tc.argv...)
			if strings.Contains(out, secret) {
				t.Fatalf("secret leaked in %v:\n%s", tc.argv, out)
			}
			if !strings.Contains(out, config.RedactedPlaceholder) {
				t.Fatalf("secret not rendered as %s in %v:\n%s",
					config.RedactedPlaceholder, tc.argv, out)
			}
		})
	}
}

func TestConfigSecretsRedactedInWrites(t *testing.T) {
	const secret = "MY_SECRET_TOKEN_VAR"
	const replacement = "OTHER_SECRET_VAR"
	f := newLayeredFixture(t, `{"registry":{"auth_token_env":"`+secret+`"}}`, "")

	out := f.runOK(t, "config", "set", "registry.auth_token_env", replacement)
	for _, leaked := range []string{secret, replacement} {
		if strings.Contains(out, leaked) {
			t.Errorf("set leaked %q:\n%s", leaked, out)
		}
	}

	out = f.runOK(t, "config", "unset", "registry.auth_token_env")
	if strings.Contains(out, replacement) {
		t.Errorf("unset leaked the removed value:\n%s", out)
	}
}

// ── §2 no duplicate config loading ──────────────────────────────────────────

// TestConfigCommandsUseInvocationSnapshot proves the read commands consume the
// Order 1 snapshot rather than loading configuration for themselves.
func TestConfigCommandsUseInvocationSnapshot(t *testing.T) {
	for _, argv := range [][]string{
		{"config", "get", "registry"},
		{"config", "get", "registry", "--scope", "effective"},
		{"config", "list"},
		{"config", "list", "--scope", "effective"},
		{"config", "explain", "registry"},
	} {
		t.Run(strings.Join(argv, "_"), func(t *testing.T) {
			env := testkit.CleanEnv(t)
			clearHelpEnv(t)
			writeUserConfig(t, env, `{"registry":"https://user.example"}`)
			p := installProbe(t)
			root := NewMRoot(testBuildInfo())
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)

			if code := runInvocation(context.Background(), root, testBuildInfo(), argv); code != 0 {
				t.Fatalf("exit=%d out=%s", code, buf.String())
			}
			if got := p.configLoads.Load(); got != 1 {
				t.Fatalf("configLoads=%d want 1 — the command loaded config again", got)
			}
		})
	}
}

// TestConfigWriteCommandsReloadSnapshotOnce pins one reload per write: the
// pre-write read comes from the bootstrap snapshot and the post-write report
// from a single republish.
func TestConfigWriteCommandsReloadSnapshotOnce(t *testing.T) {
	for _, argv := range [][]string{
		{"config", "set", "registry", "https://new.example"},
		{"config", "unset", "registry"},
	} {
		t.Run(strings.Join(argv, "_"), func(t *testing.T) {
			env := testkit.CleanEnv(t)
			clearHelpEnv(t)
			writeUserConfig(t, env, `{"registry":"https://user.example"}`)
			p := installProbe(t)
			root := NewMRoot(testBuildInfo())
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)

			if code := runInvocation(context.Background(), root, testBuildInfo(), argv); code != 0 {
				t.Fatalf("exit=%d out=%s", code, buf.String())
			}
			// One bootstrap load plus exactly one post-write reload.
			if got := p.configLoads.Load(); got != 2 {
				t.Fatalf("configLoads=%d want 2 (bootstrap + one reload)", got)
			}
		})
	}
}

// TestConfigReadCommandsHonorConfigFlagFromSnapshot checks that --config is
// interpreted once, by the snapshot, and not reinterpreted by the command.
func TestConfigReadCommandsHonorConfigFlagFromSnapshot(t *testing.T) {
	testkit.CleanEnv(t)
	clearHelpEnv(t)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "custom.jsonc")
	if err := os.WriteFile(cfg, []byte(`{"registry":"https://custom.example"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p := installProbe(t)
	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	argv := []string{"--config", cfg, "config", "get", "registry", "--scope", "effective"}
	if code := runInvocation(context.Background(), root, testBuildInfo(), argv); code != 0 {
		t.Fatalf("exit=%d out=%s", code, buf.String())
	}
	if got := strings.TrimSpace(buf.String()); got != "https://custom.example" {
		t.Fatalf("effective registry = %q, want the --config value", got)
	}
	if got := p.configLoads.Load(); got != 1 {
		t.Fatalf("configLoads=%d want 1 — --config was reinterpreted", got)
	}
}

// ── view-model unit checks ──────────────────────────────────────────────────

func TestConfigListPrefixMatchesNamespaceOnly(t *testing.T) {
	opts := configListOptions{prefix: "registry"}
	for _, key := range []string{"registry", "registry.auth_token_env"} {
		if !opts.matchesPrefix(key) {
			t.Errorf("%q must match prefix registry", key)
		}
	}
	// "registries.*" is a different namespace, not a longer "registry".
	for _, key := range []string{"registries.//npm.example/", "offline"} {
		if opts.matchesPrefix(key) {
			t.Errorf("%q must not match prefix registry", key)
		}
	}
}

func TestConfigScopeConversionRoundTrips(t *testing.T) {
	cases := map[configScope]config.Scope{
		configScopeUser:      config.ScopeUser,
		configScopeProject:   config.ScopeProject,
		configScopeEffective: config.ScopeEffective,
	}
	for cliScope, want := range cases {
		if got := configScopeToConfig(cliScope); got != want {
			t.Errorf("configScopeToConfig(%s)=%s want %s", cliScope, got, want)
		}
	}
	// Effective has no single backing layer.
	if _, ok := configScopeSource(configScopeEffective); ok {
		t.Error("effective scope must report no backing source")
	}
	if src, ok := configScopeSource(configScopeUser); !ok || src != config.SourceGlobal {
		t.Errorf("user scope source=%v ok=%v want global", src, ok)
	}
}
