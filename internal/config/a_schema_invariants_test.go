package config

import (
	"strings"
	"testing"
)

// TestRegistryInvariants enforces that every ConfigKeySpec is internally
// consistent. The registry is the single source of truth for types, defaults,
// scopes, and secrecy, so a spec that contradicts itself silently produces
// wrong validation and wrong output everywhere downstream.
func TestRegistryInvariants(t *testing.T) {
	for _, key := range RegisteredKeys() {
		spec := KeySpec(key)
		if spec == nil {
			t.Fatalf("%s: RegisteredKeys returned a key with no spec", key)
		}

		if spec.Key != key {
			t.Errorf("%s: spec.Key is %q; must match its map key", key, spec.Key)
		}
		if spec.Group == "" {
			t.Errorf("%s: Group is empty", key)
		}
		if spec.Description == "" {
			t.Errorf("%s: Description is empty", key)
		}
		if spec.Type == "" {
			t.Errorf("%s: Type is empty", key)
		}
		if key != strings.ToLower(key) {
			t.Errorf("%s: canonical keys are lowercase snake_case", key)
		}
		if strings.ContainsAny(key, " -") {
			t.Errorf("%s: canonical keys use dots and underscores only", key)
		}

		// The default must itself satisfy the spec, or a fresh install is
		// already invalid.
		if err := validateKeyValue(key, spec.Default); err != nil {
			t.Errorf("%s: default %#v fails its own validation: %v", key, spec.Default, err)
		}
		if err := validateRange(spec, spec.Default); err != nil {
			t.Errorf("%s: default %#v violates its own range: %v", key, spec.Default, err)
		}

		switch spec.Type {
		case TypeEnum:
			if len(spec.Enum) == 0 {
				t.Errorf("%s: enum type with no Enum values", key)
			}
			if s, ok := spec.Default.(string); !ok {
				t.Errorf("%s: enum default must be a string, got %T", key, spec.Default)
			} else if !containsString(spec.Enum, s) {
				t.Errorf("%s: default %q is not in Enum %v", key, s, spec.Enum)
			}
		case TypeBool:
			if _, ok := spec.Default.(bool); !ok {
				t.Errorf("%s: bool default must be bool, got %T", key, spec.Default)
			}
		case TypeInt:
			if _, ok := spec.Default.(int); !ok {
				t.Errorf("%s: int default must be int, got %T", key, spec.Default)
			}
		case TypeString:
			if _, ok := spec.Default.(string); !ok {
				t.Errorf("%s: string default must be string, got %T", key, spec.Default)
			}
		case TypeDuration:
			s, ok := spec.Default.(string)
			if !ok {
				t.Errorf("%s: duration default must be a string, got %T", key, spec.Default)
				break
			}
			if _, err := ParseDuration(s); err != nil {
				t.Errorf("%s: duration default %q does not parse: %v", key, s, err)
			}
		}

		if spec.Type != TypeEnum && len(spec.Enum) > 0 {
			t.Errorf("%s: Enum is only meaningful for enum type, got %s", key, spec.Type)
		}
		if spec.Type != TypeInt && (spec.Minimum != nil || spec.Maximum != nil) {
			t.Errorf("%s: Minimum/Maximum are only meaningful for int type, got %s", key, spec.Type)
		}
		if spec.Minimum != nil && spec.Maximum != nil && *spec.Minimum > *spec.Maximum {
			t.Errorf("%s: Minimum %d exceeds Maximum %d", key, *spec.Minimum, *spec.Maximum)
		}

		for _, s := range spec.Scopes {
			if s != ScopeUser && s != ScopeProject {
				t.Errorf("%s: %q is not a writable scope", key, s)
			}
		}
		if spec.Deprecated && spec.Replacement == "" {
			t.Errorf("%s: deprecated keys must name a Replacement", key)
		}
		if spec.Replacement != "" && !IsCanonical(spec.Replacement) {
			t.Errorf("%s: Replacement %q is not a canonical key", key, spec.Replacement)
		}
	}
}

// TestLegacyMapPointsAtCanonicalKeys keeps the legacy alias table honest: an
// alias that maps to a key the registry does not define would silently make
// the alias unusable.
func TestLegacyMapPointsAtCanonicalKeys(t *testing.T) {
	for legacy, canon := range legacyToCanonical {
		if !IsCanonical(canon) {
			t.Errorf("legacy %q maps to %q, which is not a canonical key", legacy, canon)
		}
		if IsCanonical(legacy) {
			t.Errorf("legacy %q is also a canonical key; one spelling must win", legacy)
		}
	}
}

// TestEnvVarMapPointsAtCanonicalKeys ensures no env var can set a key the
// registry does not know, which would bypass type validation.
func TestEnvVarMapPointsAtCanonicalKeys(t *testing.T) {
	for key, envVar := range envVarByKey {
		if !IsCanonical(key) {
			t.Errorf("env var %s targets %q, which is not a canonical key", envVar, key)
		}
		if err := validateEnvVarName(envVar); err != nil {
			t.Errorf("env var name %q is not a valid variable name: %v", envVar, err)
		}
	}
}

// TestAllowedValuesMatchesRegistry checks the derived display table cannot
// drift from the specs it is built from.
func TestAllowedValuesMatchesRegistry(t *testing.T) {
	for _, key := range RegisteredKeys() {
		spec := KeySpec(key)
		got := AllowedValues(key)
		switch spec.Type {
		case TypeEnum:
			want := strings.Join(spec.Enum, "|")
			if got != want {
				t.Errorf("%s: AllowedValues = %q, want %q", key, got, want)
			}
		case TypeBool:
			if got != "true|false" {
				t.Errorf("%s: AllowedValues = %q, want \"true|false\"", key, got)
			}
		default:
			if got != "" {
				t.Errorf("%s: AllowedValues = %q, want empty for free-form %s", key, got, spec.Type)
			}
		}
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
