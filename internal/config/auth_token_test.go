package config_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
)

func TestAuthTokenFromSnapshot(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	cfgPath := filepath.Join(root, "m.jsonc")
	if err := config.SetFile(cfgPath, "registry.auth_token_env", "NPM_TOKEN"); err != nil {
		t.Fatal(err)
	}
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:                  root,
		ProjectRoot:          root,
		ProjectPath:          cfgPath,
		RequireProjectConfig: true,
		Env:                  []string{"NPM_TOKEN=invocation-secret"},
		EnvSnapshot:          config.NewEnvSnapshot([]string{"NPM_TOKEN=invocation-secret"}, "linux"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := config.AuthToken(eff); got != "invocation-secret" {
		t.Fatalf("token=%q want invocation-secret", got)
	}
}

func TestAuthTokenEmptyEnvNoToken(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NPM_TOKEN", "host-secret")
	root := t.TempDir()
	cfgPath := filepath.Join(root, "m.jsonc")
	if err := config.SetFile(cfgPath, "registry.auth_token_env", "NPM_TOKEN"); err != nil {
		t.Fatal(err)
	}
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:                  root,
		ProjectRoot:          root,
		ProjectPath:          cfgPath,
		RequireProjectConfig: true,
		Env:                  []string{},
		EnvSnapshot:          config.NewEnvSnapshot([]string{}, "linux"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := config.AuthToken(eff); got != "" {
		t.Fatalf("token=%q want empty", got)
	}
}

func TestAuthTokenWindowsCaseInsensitive(t *testing.T) {
	snap := config.NewEnvSnapshot([]string{"npm_token=win-invocation"}, "windows")
	eff := &config.Effective{
		Values: map[string]config.Value{
			"registry.auth_token_env": {Raw: "NPM_TOKEN"},
		},
		Env: snap,
	}
	if got := config.AuthToken(eff); got != "win-invocation" {
		t.Fatalf("token=%q want win-invocation", got)
	}
}

func TestAuthTokenUninitializedFallsBackAmbient(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NPM_TOKEN", "ambient-fallback")
	eff := &config.Effective{
		Values: map[string]config.Value{
			"registry.auth_token_env": {Raw: "NPM_TOKEN"},
		},
	}
	if got := config.AuthToken(eff); got != "ambient-fallback" {
		t.Fatalf("token=%q want ambient-fallback", got)
	}
}
