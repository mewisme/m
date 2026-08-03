package lifecycle_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/lifecycle"
)

func TestScriptTimeoutDefault(t *testing.T) {
	got, err := lifecycle.ScriptTimeout(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != 10*time.Minute {
		t.Fatalf("default timeout=%v", got)
	}
}

func TestScriptTimeoutFromConfigIgnoresAmbient(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	cfgPath := filepath.Join(root, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"lifecycle":{"script_timeout":"1s"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	eff, err := config.Load(ctx, config.LoadOptions{
		CWD:         root,
		ProjectRoot: root,
		ProjectPath: cfgPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEW_LIFECYCLE_SCRIPT_TIMEOUT", "10m")
	got, err := lifecycle.ScriptTimeout(eff)
	if err != nil {
		t.Fatal(err)
	}
	if got != time.Second {
		t.Fatalf("timeout=%v want 1s from loaded config", got)
	}
}

func TestScriptTimeoutExplicitEmptyUsesDefault(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"lifecycle.script_timeout": {Raw: ""},
	}}
	got, err := lifecycle.ScriptTimeout(eff)
	if err != nil {
		t.Fatal(err)
	}
	if got != 10*time.Minute {
		t.Fatalf("empty config should use default, got %v", got)
	}
}

func TestScriptTimeoutInvalidConfig(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"lifecycle.script_timeout": {Raw: "not-a-duration"},
	}}
	_, err := lifecycle.ScriptTimeout(eff)
	if err == nil || apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("expected config error, got %v", err)
	}
}

func TestScriptTimeoutAfterAppNewIgnoresPostAmbient(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"t","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"lifecycle":{"script_timeout":"2s"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ac, err := app.New(ctx, app.Options{CWD: root, ConfigPath: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEW_LIFECYCLE_SCRIPT_TIMEOUT", "10m")
	got, err := lifecycle.ScriptTimeout(ac.Config)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2*time.Second {
		t.Fatalf("post-app ambient env leaked: got %v", got)
	}
}
