package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errCodes lists the error-severity codes a result carries, in report order.
func errCodes(res ValidationResult) []DiagnosticCode {
	var out []DiagnosticCode
	for _, d := range res.Errors() {
		out = append(out, d.Code)
	}
	return out
}

// ── §3 integer ranges ───────────────────────────────────────────────────────

// TestValidateIntegerRanges pins the three range-carrying int keys. Values read
// from a file arrive as json.Number, which is the shape that previously slipped
// past every range check.
func TestValidateIntegerRanges(t *testing.T) {
	rangedKeys := []string{
		"resolve.minimum_release_age",
		"transaction.snapshot_retention",
		"runner.mx.cache.retention_days",
	}
	for _, key := range rangedKeys {
		spec := KeySpec(key)
		if spec == nil || spec.Minimum == nil {
			t.Fatalf("%s must declare a Minimum for the range check to mean anything", key)
		}
		t.Run(key+"/below_minimum", func(t *testing.T) {
			res := ValidateDocument(nestedDoc(t, key, -1), "test.jsonc", ValidateOptions{})
			if res.Valid() {
				t.Fatalf("%s = -1 accepted; schema minimum is %d", key, *spec.Minimum)
			}
			if !hasCode(res, DiagConstraint) {
				t.Errorf("want constraint diagnostic, got %+v", res.Diagnostics)
			}
		})
		t.Run(key+"/at_minimum", func(t *testing.T) {
			res := ValidateDocument(nestedDoc(t, key, *spec.Minimum), "test.jsonc", ValidateOptions{})
			if !res.Valid() {
				t.Errorf("%s = %d rejected at its own minimum: %+v", key, *spec.Minimum, res.Diagnostics)
			}
		})
	}
}

// TestValidateRangeAboveMaximum covers the Maximum branch. No live key declares
// a maximum, so the check is exercised against the spec directly rather than
// inventing a registry entry the product does not have.
func TestValidateRangeAboveMaximum(t *testing.T) {
	var max int64 = 5
	spec := &ConfigKeySpec{Key: "x", Type: TypeInt, Maximum: &max}

	if err := validateRange(spec, json.Number("6")); err == nil {
		t.Error("6 accepted above maximum 5")
	}
	if err := validateRange(spec, json.Number("5")); err != nil {
		t.Errorf("5 rejected at its own maximum: %v", err)
	}
	// A file-sourced value is a json.Number; an int is what a Go caller passes.
	if err := validateRange(spec, 6); err == nil {
		t.Error("int 6 accepted above maximum 5")
	}
}

// nestedDoc builds a JSONC document setting a dotted key to v, in the nested
// shape a user actually writes.
func nestedDoc(t *testing.T, key string, v any) []byte {
	t.Helper()
	parts := strings.Split(key, ".")
	doc := any(v)
	for i := len(parts) - 1; i >= 0; i-- {
		doc = map[string]any{parts[i]: doc}
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal %s: %v", key, err)
	}
	return b
}

// ── §4 scopes ───────────────────────────────────────────────────────────────

// TestValidateUserOnlyKeysRejectedInProject: every user-only key must be
// refused in a project document, and the message must name the key, the
// rejected scope, and the scopes the key does allow.
func TestValidateUserOnlyKeysRejectedInProject(t *testing.T) {
	userOnly := []string{
		"cache.dir", "store.dir", "link.use_global_store",
		"runner.mx.cache.dir", "ui.theme", "ui.pager",
	}
	for _, key := range userOnly {
		t.Run(key, func(t *testing.T) {
			spec := KeySpec(key)
			if spec == nil {
				t.Fatalf("%s is not registered", key)
			}
			if len(spec.Scopes) != 1 || spec.Scopes[0] != ScopeUser {
				t.Skipf("%s is no longer user-only (scopes=%v)", key, spec.Scopes)
			}
			res := ValidateDocument(nestedDoc(t, key, sampleValue(spec)), "m.jsonc",
				ValidateOptions{Scope: ScopeProject})
			if res.Valid() {
				t.Fatalf("%s accepted in project scope: %+v", key, res.Diagnostics)
			}
			d := findCode(t, res, DiagScope)
			if d.Key != key {
				t.Errorf("diagnostic key = %q, want %q", d.Key, key)
			}
			for _, want := range []string{"project", "user"} {
				if !strings.Contains(d.Message, want) {
					t.Errorf("message %q should name %q", d.Message, want)
				}
			}
		})
	}
}

// TestValidateUserOnlyKeyAllowedInUserScope is the other half: the same key in
// its own scope is fine, so the scope check is not simply rejecting everything.
func TestValidateUserOnlyKeyAllowedInUserScope(t *testing.T) {
	res := ValidateDocument([]byte(`{"cache": {"dir": "/tmp/cache"}}`), "config.jsonc",
		ValidateOptions{Scope: ScopeUser})
	if !res.Valid() {
		t.Errorf("cache.dir rejected in user scope: %+v", res.Diagnostics)
	}
}

// TestValidateEffectiveScopeSkipsScopeCheck: effective is not a writable
// document scope, so it must not apply per-key scope restrictions.
func TestValidateEffectiveScopeSkipsScopeCheck(t *testing.T) {
	res := ValidateDocument([]byte(`{"cache": {"dir": "/tmp/cache"}}`), "merged",
		ValidateOptions{Scope: ScopeEffective})
	if !res.Valid() {
		t.Errorf("effective scope applied a writable-scope check: %+v", res.Diagnostics)
	}
}

// TestLoadDoesNotApplyScopeRules: the writable-scope rule governs where
// `m config set` may write a key, not whether an existing file may be read. A
// project file already carrying a user-only key must keep loading, or a rule
// about writes would break working installs.
func TestLoadDoesNotApplyScopeRules(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(proj, []byte(`{"link": {"use_global_store": true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	eff, err := Load(context.Background(), LoadOptions{
		CWD:         dir,
		ProjectRoot: dir,
		ProjectPath: proj,
		GlobalPath:  filepath.Join(dir, "absent.jsonc"),
		Env:         []string{},
	})
	if err != nil {
		t.Fatalf("user-only key in a project file blocked loading: %v", err)
	}
	if v, ok := eff.Values["link.use_global_store"]; !ok || v.Raw != true {
		t.Errorf("value not retained: %+v", eff.Values["link.use_global_store"])
	}
	// The same document still reports the placement when asked in project scope,
	// which is where the user goes to hear about it.
	res := ValidateDocument([]byte(`{"link": {"use_global_store": true}}`), proj,
		ValidateOptions{Scope: ScopeProject})
	if res.Valid() {
		t.Error("config validate --scope project should still flag a user-only key")
	}
}

func sampleValue(spec *ConfigKeySpec) any {
	switch spec.Type {
	case TypeBool:
		return true
	case TypeInt:
		return 1
	case TypeEnum:
		if len(spec.Enum) > 0 {
			return spec.Enum[0]
		}
		return "x"
	case TypeDuration:
		return "1s"
	default:
		return "value"
	}
}

func findCode(t *testing.T, res ValidationResult, code DiagnosticCode) Diagnostic {
	t.Helper()
	for _, d := range res.Diagnostics {
		if d.Code == code {
			return d
		}
	}
	t.Fatalf("no %s diagnostic in %+v", code, res.Diagnostics)
	return Diagnostic{}
}

// ── §5 legacy keys and conflicts ────────────────────────────────────────────

// TestValidateNestedLegacyConflict is the case a root-map lookup on a dotted
// string can never find: both spellings are written nested, so detection has to
// work on flattened paths.
func TestValidateNestedLegacyConflict(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		legacy string
		canon  string
	}{
		{
			name:   "resolve.autoInstallPeers",
			src:    `{"resolve": {"autoInstallPeers": true, "auto_install_peers": false}}`,
			legacy: "resolve.autoInstallPeers",
			canon:  "resolve.auto_install_peers",
		},
		{
			name:   "network.timeout_ms",
			src:    `{"network": {"timeout_ms": 1000, "timeout": "1s"}}`,
			legacy: "network.timeout_ms",
			canon:  "network.timeout",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := ValidateDocument([]byte(tc.src), "test.jsonc", ValidateOptions{})
			if res.Valid() {
				t.Fatalf("conflict not detected: %+v", res.Diagnostics)
			}
			d := findCode(t, res, DiagConflictingKey)
			if d.Key != tc.canon {
				t.Errorf("canonical key = %q, want %q", d.Key, tc.canon)
			}
			if d.LegacyKey != tc.legacy {
				t.Errorf("legacy key = %q, want %q", d.LegacyKey, tc.legacy)
			}
			// A conflict is not also a "use the canonical name" nudge: the user
			// wrote both, so there is nothing to rename.
			if hasCode(res, DiagLegacyKey) {
				t.Errorf("conflict should not also emit a legacy-key warning: %+v", res.Diagnostics)
			}
		})
	}
}

// TestValidateNestedLegacyAloneWarns: the nested legacy spelling on its own is
// a warning that names its replacement.
func TestValidateNestedLegacyAloneWarns(t *testing.T) {
	res := ValidateDocument([]byte(`{"resolve": {"autoInstallPeers": true}}`),
		"test.jsonc", ValidateOptions{})
	if !res.Valid() {
		t.Fatalf("lone legacy spelling should be valid: %+v", res.Diagnostics)
	}
	d := findCode(t, res, DiagLegacyKey)
	if d.Severity != SeverityWarning {
		t.Errorf("severity = %s, want warning", d.Severity)
	}
	if d.Replacement != "resolve.auto_install_peers" {
		t.Errorf("replacement = %q, want resolve.auto_install_peers", d.Replacement)
	}
	if d.ReportedKey() != "resolve.autoInstallPeers" {
		t.Errorf("reported key = %q; the report must name what the user typed", d.ReportedKey())
	}
}

// TestValidateEveryLegacyKeyResolves: each registered legacy spelling warns and
// points at a canonical key that actually exists.
func TestValidateEveryLegacyKeyResolves(t *testing.T) {
	for legacy, canon := range legacyToCanonical {
		t.Run(legacy, func(t *testing.T) {
			spec := KeySpec(canon)
			if spec == nil {
				t.Fatalf("legacy %s maps to unregistered %s", legacy, canon)
			}
			res := ValidateDocument(nestedDoc(t, legacy, sampleValue(spec)),
				"test.jsonc", ValidateOptions{})
			if !res.Valid() {
				t.Fatalf("legacy %s rejected: %+v", legacy, res.Diagnostics)
			}
			d := findCode(t, res, DiagLegacyKey)
			if d.Replacement != canon {
				t.Errorf("replacement = %q, want %q", d.Replacement, canon)
			}
		})
	}
}

// ── §1 deterministic ordering ───────────────────────────────────────────────

// TestValidateDiagnosticOrderIsDeterministic: repeated validation of the same
// bytes must produce byte-identical reports, and the order must be by key.
func TestValidateDiagnosticOrderIsDeterministic(t *testing.T) {
	// Several findings across unrelated keys; map iteration would otherwise
	// shuffle them between runs.
	src := []byte(`{
		"offline": "yes",
		"install": {"linker": "magic", "nope": true},
		"network": {"timeout": "soon"},
		"transaction": {"snapshot_retention": -1},
		"prefer-offline": true
	}`)

	first := ValidateDocument(src, "test.jsonc", ValidateOptions{})
	if len(first.Diagnostics) < 5 {
		t.Fatalf("expected several diagnostics, got %+v", first.Diagnostics)
	}
	want, err := json.Marshal(first.Diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		got, err := json.Marshal(ValidateDocument(src, "test.jsonc", ValidateOptions{}).Diagnostics)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("run %d differs:\n got %s\nwant %s", i, got, want)
		}
	}
	// Ordering is by reported key, so the report reads in key order.
	prev := ""
	for _, d := range first.Diagnostics {
		if k := d.ReportedKey(); k < prev {
			t.Errorf("diagnostics not ordered by key: %q after %q", k, prev)
		} else {
			prev = k
		}
	}
}

// ── §10 redaction ───────────────────────────────────────────────────────────

// TestValidateRedactsSecretValues: a bad value for a secret key must not appear
// in any diagnostic, and neither must a token pasted where an env var name
// belongs.
func TestValidateRedactsSecretValues(t *testing.T) {
	const token = "npm_aBcDeF1234567890secret"
	res := ValidateDocument([]byte(`{"registry": {"auth_token_env": "`+token+`"}}`),
		"config.jsonc", ValidateOptions{})
	if res.Valid() {
		t.Fatalf("inline secret accepted: %+v", res.Diagnostics)
	}
	for _, d := range res.Diagnostics {
		if strings.Contains(d.Message, token) {
			t.Errorf("diagnostic leaked the secret: %q", d.Message)
		}
	}
	// The error the loader raises must be equally clean.
	err := res.Err()
	if err == nil {
		t.Fatal("Err() returned nil for an invalid document")
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("load error leaked the secret: %v", err)
	}
	if !strings.Contains(err.Error(), "registry.auth_token_env") {
		t.Errorf("load error should name the key: %v", err)
	}
}

// TestRedactDiagnosticMessageWithholdsSecretValues covers the redaction
// boundary for a secret key whose value fails an ordinary type check.
func TestRedactDiagnosticMessageWithholdsSecretValues(t *testing.T) {
	for _, key := range RegisteredKeys() {
		if !KeySpec(key).Secret {
			continue
		}
		got := redactDiagnosticMessage(key, `expected string, got "hunter2"`)
		if strings.Contains(got, "hunter2") {
			t.Errorf("%s: message kept the value: %q", key, got)
		}
	}
	// Non-secret keys keep their detail: withholding everything would make
	// diagnostics useless.
	got := redactDiagnosticMessage("registry", `expected string, got 42`)
	if !strings.Contains(got, "42") {
		t.Errorf("non-secret message was redacted: %q", got)
	}
}

// ── ValidationResult.Err bridge ─────────────────────────────────────────────

// TestValidationResultErr: the bridge that turns a report into the typed error
// configuration loading returns.
func TestValidationResultErr(t *testing.T) {
	if err := ValidateDocument([]byte(`{"offline": true}`), "ok.jsonc", ValidateOptions{}).Err(); err != nil {
		t.Errorf("valid document produced an error: %v", err)
	}

	err := ValidateDocument([]byte(`{"install": {"linker": "magic"}}`), "bad.jsonc", ValidateOptions{}).Err()
	if err == nil {
		t.Fatal("invalid document produced no error")
	}
	for _, want := range []string{"ERR_M_CONFIG", "bad.jsonc", "install.linker"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}

	// A warning-only document still loads.
	if err := ValidateDocument([]byte(`{"prefer-offline": true}`), "legacy.jsonc", ValidateOptions{}).Err(); err != nil {
		t.Errorf("warning-only document blocked loading: %v", err)
	}
}

// ── report aggregation ──────────────────────────────────────────────────────

// TestValidateFilesAggregates: every file is validated even when an earlier one
// is broken, absent files contribute nothing, and counts sum across files.
func TestValidateFilesAggregates(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "config.jsonc")
	projPath := filepath.Join(dir, "m.jsonc")
	absent := filepath.Join(dir, "gone.jsonc")

	// Broken user file, and a project file with a distinct problem after it.
	if err := os.WriteFile(userPath, []byte(`{"offline": `), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projPath, []byte(`{"install": {"linker": "magic"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	rep := ValidateFiles(ScopeEffective,
		[]string{userPath, projPath, absent},
		[]Scope{ScopeUser, ScopeProject, ScopeProject},
		ValidateOptions{})

	if rep.Valid {
		t.Fatal("report claims valid despite two broken files")
	}
	if len(rep.Files) != 2 {
		t.Fatalf("files = %d, want 2 (the absent file must not be listed)", len(rep.Files))
	}
	if rep.Files[0].Path != userPath || rep.Files[1].Path != projPath {
		t.Errorf("files out of resolution order: %s then %s", rep.Files[0].Path, rep.Files[1].Path)
	}
	if rep.Files[0].Scope != ScopeUser || rep.Files[1].Scope != ScopeProject {
		t.Errorf("per-file scopes wrong: %s, %s", rep.Files[0].Scope, rep.Files[1].Scope)
	}
	// The broken first file did not stop the second from being checked.
	if !hasCode(rep.Files[0], DiagSyntax) {
		t.Errorf("user file: want syntax error, got %+v", rep.Files[0].Diagnostics)
	}
	if !hasCode(rep.Files[1], DiagConstraint) {
		t.Errorf("project file: want constraint error, got %+v", rep.Files[1].Diagnostics)
	}
	if len(rep.Errors()) < 2 {
		t.Errorf("aggregate errors = %d, want at least 2", len(rep.Errors()))
	}
}

// TestValidateFilesAllAbsentIsValid: no config at all is a legal state.
func TestValidateFilesAllAbsentIsValid(t *testing.T) {
	dir := t.TempDir()
	rep := ValidateFiles(ScopeEffective,
		[]string{filepath.Join(dir, "a.jsonc"), filepath.Join(dir, "b.jsonc")},
		[]Scope{ScopeUser, ScopeProject}, ValidateOptions{})
	if !rep.Valid {
		t.Errorf("absent files reported invalid: %+v", rep.Diagnostics())
	}
	if len(rep.Files) != 0 || rep.KeyCount() != 0 {
		t.Errorf("files=%d keys=%d, want 0 and 0", len(rep.Files), rep.KeyCount())
	}
}

// TestValidateFilesWarningsStayValid: warnings alone do not invalidate a
// report, and --strict is what promotes them.
func TestValidateFilesWarningsStayValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.jsonc")
	if err := os.WriteFile(path, []byte(`{"prefer-offline": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	lenient := ValidateFiles(ScopeUser, []string{path}, []Scope{ScopeUser}, ValidateOptions{})
	if !lenient.Valid {
		t.Errorf("warning made the report invalid: %+v", lenient.Diagnostics())
	}
	if len(lenient.Warnings()) != 1 {
		t.Errorf("warnings = %d, want 1", len(lenient.Warnings()))
	}
	if lenient.KeyCount() != 1 {
		t.Errorf("keys = %d, want 1", lenient.KeyCount())
	}

	strict := ValidateFiles(ScopeUser, []string{path}, []Scope{ScopeUser},
		ValidateOptions{Strict: true})
	if strict.Valid {
		t.Error("--strict did not promote the legacy warning to an error")
	}
}

// TestValidateFilesUnreadableFileIsDiagnosed: a file that exists but cannot be
// read is a reported finding, not a silent pass. Directories stand in for an
// unreadable path because they are refused portably.
func TestValidateFilesUnreadableFileIsDiagnosed(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "config.jsonc")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	rep := ValidateFiles(ScopeUser, []string{blocked}, []Scope{ScopeUser}, ValidateOptions{})
	if rep.Valid {
		t.Fatal("unreadable file reported valid")
	}
	if !hasCode(rep.Files[0], DiagRead) && !hasCode(rep.Files[0], DiagSyntax) {
		t.Errorf("want a read or syntax diagnostic, got %+v", rep.Files[0].Diagnostics)
	}
}

// TestValidateFilesUnscopedTailDoesNotInheritScope: a short scopes slice must
// leave the remaining files unscoped rather than reusing the last scope.
func TestValidateFilesUnscopedTailDoesNotInheritScope(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	// cache.dir is user-only, so it would fail if the project scope leaked in.
	if err := os.WriteFile(path, []byte(`{"cache": {"dir": "/tmp/c"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rep := ValidateFiles(ScopeEffective, []string{path}, nil, ValidateOptions{})
	if !rep.Valid {
		t.Errorf("unscoped file got a scope check: %+v", rep.Diagnostics())
	}
}

// ── §2 whole-document coverage the table above does not reach ───────────────

// TestValidateUnknownKeyInOwnedNamespace: a typo inside a namespace Mew owns is
// an error, and the diagnostic names the key as written.
//
// A key outside every owned namespace is deliberately NOT an error: the format
// stays open for keys other tools write. The boundary is the namespace, not the
// registry.
func TestValidateUnknownKeyInOwnedNamespace(t *testing.T) {
	res := ValidateDocument([]byte(`{"install": {"linkr": "hoisted"}}`), "test.jsonc", ValidateOptions{})
	if res.Valid() {
		t.Fatalf("unknown key in an owned namespace accepted: %+v", res.Diagnostics)
	}
	d := findCode(t, res, DiagUnknownKey)
	if d.Key != "install.linkr" {
		t.Errorf("diagnostic key = %q, want install.linkr", d.Key)
	}

	// Same document shape, unowned namespace: left alone.
	free := ValidateDocument([]byte(`{"someOtherTool": {"setting": 1}}`), "test.jsonc", ValidateOptions{})
	if !free.Valid() {
		t.Errorf("key outside every owned namespace rejected: %+v", free.Diagnostics)
	}
}

// TestValidateBooleanAndStringTypes: the plain type checks, including the
// string-where-bool-expected case a JSONC file makes easy to write.
func TestValidateBooleanAndStringTypes(t *testing.T) {
	cases := []struct {
		src  string
		code DiagnosticCode
	}{
		{`{"offline": "true"}`, DiagType},   // quoted bool
		{`{"offline": 1}`, DiagType},        // numeric bool
		{`{"registry": true}`, DiagType},    // bool where string expected
		{`{"registry": 42}`, DiagType},      // number where string expected
		{`{"network": {"timeout": 5}}`, ""}, // int for a duration: shape-dependent
	}
	for _, tc := range cases {
		res := ValidateDocument([]byte(tc.src), "test.jsonc", ValidateOptions{})
		if tc.code == "" {
			continue
		}
		if res.Valid() {
			t.Errorf("%s accepted: %+v", tc.src, res.Diagnostics)
			continue
		}
		if !hasCode(res, tc.code) {
			t.Errorf("%s: want %s, got %v", tc.src, tc.code, errCodes(res))
		}
	}
}

// TestValidateDynamicRegistriesNamespace: scoped registry entries are
// free-form, but the value still has to be a string.
func TestValidateDynamicRegistriesNamespace(t *testing.T) {
	ok := ValidateDocument([]byte(`{"registries": {"@acme": "https://acme.example", "@other": "https://o.example"}}`),
		"test.jsonc", ValidateOptions{})
	if !ok.Valid() {
		t.Errorf("scoped registries rejected: %+v", ok.Diagnostics)
	}
	bad := ValidateDocument([]byte(`{"registries": {"@acme": true}}`), "test.jsonc", ValidateOptions{})
	if bad.Valid() {
		t.Errorf("non-string registry URL accepted: %+v", bad.Diagnostics)
	}
}
