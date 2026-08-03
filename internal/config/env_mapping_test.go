package config_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/config"
)

// Every environment mapping must name a real registry key. A typo here would
// otherwise create a variable that silently does nothing.
func TestEnvVarKeysAreRegistryKeys(t *testing.T) {
	for _, key := range config.EnvVarKeys() {
		if config.KeySpec(key) == nil {
			t.Errorf("env-mapped key %q is not in the key registry", key)
		}
		if config.EnvVar(key) == "" {
			t.Errorf("key %q has an empty env var name", key)
		}
	}
}

// Every declared mapping must actually reach the effective config. This is the
// check that catches a key being mapped but never applied.
func TestEveryEnvMappingApplies(t *testing.T) {
	for _, key := range config.EnvVarKeys() {
		spec := config.KeySpec(key)
		if spec == nil {
			continue // covered by TestEnvVarKeysAreRegistryKeys
		}
		val := sampleEnvValue(t, spec)
		if val == "" {
			continue
		}
		envVar := config.EnvVar(key)
		dir := t.TempDir()
		eff, err := config.Load(context.Background(), config.LoadOptions{
			CWD:         dir,
			ProjectRoot: dir,
			GlobalPath:  filepath.Join(dir, "no-global.jsonc"),
			Env:         []string{envVar + "=" + val},
			IdentityMew: true,
		})
		if err != nil {
			t.Errorf("%s=%s: load failed: %v", envVar, val, err)
			continue
		}
		got, err := config.Get(eff, key)
		if err != nil {
			t.Errorf("%s: %v", key, err)
			continue
		}
		if got.Source != config.SourceEnv {
			t.Errorf("%s via %s: source=%s want env (mapping declared but not applied)",
				key, envVar, got.Source)
		}
		if got.Path != envVar {
			t.Errorf("%s via %s: path=%q want %q", key, envVar, got.Path, envVar)
		}
	}
}

// A malformed environment value must fail closed, not be silently dropped.
func TestEnvInvalidValueFailsClosed(t *testing.T) {
	dir := t.TempDir()
	_, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         dir,
		ProjectRoot: dir,
		GlobalPath:  filepath.Join(dir, "no-global.jsonc"),
		Env:         []string{"MEW_OFFLINE=maybe"},
		IdentityMew: true,
	})
	if err == nil {
		t.Fatal("expected malformed MEW_OFFLINE to fail")
	}
	if !strings.Contains(err.Error(), "MEW_OFFLINE") {
		t.Fatalf("error should name the variable: %v", err)
	}
}

// sampleEnvValue returns a value valid for the key's declared type.
func sampleEnvValue(t *testing.T, spec *config.ConfigKeySpec) string {
	t.Helper()
	switch spec.Type {
	case config.TypeBool:
		return "true"
	case config.TypeInt:
		return "7"
	case config.TypeDuration:
		return "30s"
	case config.TypeEnum:
		if len(spec.Enum) == 0 {
			return ""
		}
		for _, e := range spec.Enum {
			if e != "" {
				return e
			}
		}
		return ""
	case config.TypeString:
		if spec.Key == "registry.auth_token_env" {
			return "NPM_TOKEN" // must be a var name, not a secret
		}
		if strings.HasSuffix(spec.Key, ".dir") {
			return t.TempDir()
		}
		return "sample-value"
	default:
		return ""
	}
}
