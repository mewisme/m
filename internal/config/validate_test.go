package config

import (
	"testing"
)

// hasCode reports whether the result carries a diagnostic with the given code.
func hasCode(res ValidationResult, code DiagnosticCode) bool {
	for _, d := range res.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

func codeSeverity(res ValidationResult, code DiagnosticCode) (Severity, bool) {
	for _, d := range res.Diagnostics {
		if d.Code == code {
			return d.Severity, true
		}
	}
	return "", false
}

func TestValidateDocument(t *testing.T) {
	cases := []struct {
		name      string
		src       string
		opts      ValidateOptions
		wantCode  DiagnosticCode
		wantValid bool
	}{
		{
			name:      "clean document",
			src:       `{"registry": "https://registry.npmjs.org", "offline": true}`,
			wantValid: true,
		},
		{
			name:     "broken jsonc",
			src:      `{"registry": }`,
			wantCode: DiagSyntax,
		},
		{
			name:     "root is not an object",
			src:      `[1, 2, 3]`,
			wantCode: DiagRoot,
		},
		{
			name:     "unknown key in an owned namespace",
			src:      `{"install": {"nope": true}}`,
			wantCode: DiagUnknownKey,
		},
		{
			name:     "wrong type",
			src:      `{"offline": "yes"}`,
			wantCode: DiagType,
		},
		{
			name:     "enum violation",
			src:      `{"install": {"linker": "magic"}}`,
			wantCode: DiagConstraint,
		},
		{
			name:     "duration does not parse",
			src:      `{"network": {"timeout": "soon"}}`,
			wantCode: DiagConstraint,
		},
		{
			name:     "below minimum",
			src:      `{"transaction": {"snapshot_retention": -1}}`,
			wantCode: DiagConstraint,
		},
		{
			name:     "duplicate key",
			src:      `{"offline": true, "offline": false}`,
			wantCode: DiagDuplicateKey,
		},
		{
			name:     "secret pasted instead of env var name",
			src:      `{"registry": {"auth_token_env": "npm_aBcDeF123456"}}`,
			wantCode: DiagSecret,
		},
		{
			name:     "key not writable in project scope",
			src:      `{"cache": {"dir": "/tmp/cache"}}`,
			opts:     ValidateOptions{Scope: ScopeProject},
			wantCode: DiagScope,
		},
		{
			name:     "legacy and canonical together",
			src:      `{"network": {"timeout_ms": 1000, "timeout": "1s"}}`,
			wantCode: DiagConflictingKey,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := ValidateDocument([]byte(tc.src), "test.jsonc", tc.opts)
			if tc.wantValid {
				if !res.Valid() {
					t.Fatalf("want valid, got diagnostics: %+v", res.Diagnostics)
				}
				return
			}
			if res.Valid() {
				t.Fatalf("want invalid with %s, got no error diagnostics: %+v", tc.wantCode, res.Diagnostics)
			}
			if !hasCode(res, tc.wantCode) {
				t.Errorf("missing %s; got %+v", tc.wantCode, res.Diagnostics)
			}
		})
	}
}

// TestValidateLegacyKeySeverity: a legacy spelling still works, so it warns by
// default and only fails under --strict.
func TestValidateLegacyKeySeverity(t *testing.T) {
	src := []byte(`{"prefer-offline": true}`)

	lenient := ValidateDocument(src, "test.jsonc", ValidateOptions{})
	if !lenient.Valid() {
		t.Errorf("legacy key should be valid by default, got %+v", lenient.Diagnostics)
	}
	sev, ok := codeSeverity(lenient, DiagLegacyKey)
	if !ok || sev != SeverityWarning {
		t.Errorf("legacy severity = %v (present=%v), want warning", sev, ok)
	}

	strict := ValidateDocument(src, "test.jsonc", ValidateOptions{Strict: true})
	if strict.Valid() {
		t.Errorf("legacy key should fail under --strict, got %+v", strict.Diagnostics)
	}
}

// TestValidateAllowsFreeFormNamespace keeps registries.* scope entries legal.
func TestValidateAllowsFreeFormNamespace(t *testing.T) {
	res := ValidateDocument([]byte(`{"registries": {"@acme": "https://acme.example"}}`),
		"test.jsonc", ValidateOptions{})
	if !res.Valid() {
		t.Errorf("registries.* should be free-form, got %+v", res.Diagnostics)
	}
}

// TestValidateMissingFileIsValid: an absent config is a legal state.
func TestValidateMissingFileIsValid(t *testing.T) {
	res := ValidateFile(t.TempDir()+"/absent.jsonc", ValidateOptions{})
	if !res.Valid() || len(res.Diagnostics) != 0 {
		t.Errorf("absent file should be valid with no diagnostics, got %+v", res.Diagnostics)
	}
}

// TestValidateCountsKeys backs the "N keys checked" figure the CLI prints.
func TestValidateCountsKeys(t *testing.T) {
	res := ValidateDocument([]byte(`{"offline": true, "network": {"proxy": "", "timeout": "1s"}}`),
		"test.jsonc", ValidateOptions{})
	if res.KeyCount != 3 {
		t.Errorf("KeyCount = %d, want 3", res.KeyCount)
	}
}

// TestDurationValueReportsFailures: the typed accessor must not silently
// substitute a default the way Duration does.
func TestDurationValueReportsFailures(t *testing.T) {
	eff := &Effective{Values: map[string]Value{
		"network.timeout": {Raw: "45s", Source: SourceProject},
		"offline":         {Raw: true, Source: SourceProject},
	}}

	d, err := DurationValue(eff, "network.timeout")
	if err != nil {
		t.Fatalf("DurationValue: %v", err)
	}
	if d.String() != "45s" {
		t.Errorf("duration = %v, want 45s", d)
	}

	if _, err := DurationValue(eff, "offline"); err == nil {
		t.Error("DurationValue on a bool key succeeded; want a type error")
	}
	if _, err := DurationValue(eff, "no.such.key"); err == nil {
		t.Error("DurationValue on an unknown key succeeded; want an error")
	}

	bad := &Effective{Values: map[string]Value{
		"network.timeout": {Raw: "soon", Source: SourceProject},
	}}
	if _, err := DurationValue(bad, "network.timeout"); err == nil {
		t.Error("DurationValue on an unparseable value succeeded; want an error")
	}
}

// TestRedactValueCoversSecretKeys pins the single redaction boundary.
func TestRedactValueCoversSecretKeys(t *testing.T) {
	if got := RedactValue("registry.auth_token_env", "NPM_TOKEN"); got != RedactedPlaceholder {
		t.Errorf("secret key rendered as %v, want %q", got, RedactedPlaceholder)
	}
	if got := RedactValue("registry", "https://registry.npmjs.org"); got != "https://registry.npmjs.org" {
		t.Errorf("non-secret key was altered: %v", got)
	}
	// Every registry key marked Secret must redact, so adding one cannot
	// accidentally bypass the boundary.
	for _, key := range RegisteredKeys() {
		if !KeySpec(key).Secret {
			continue
		}
		if RedactValue(key, "sensitive") != RedactedPlaceholder {
			t.Errorf("%s is Secret but RedactValue passed the value through", key)
		}
	}
}
