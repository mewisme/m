package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
)

func TestPrecedenceEnvOverProjectOverGlobal(t *testing.T) {
	root := testkit.ModuleRoot(t)
	prec := filepath.Join(root, "fixtures", "config", "precedence")
	global := filepath.Join(prec, "global", "config.jsonc")

	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         prec,
		ProjectRoot: prec,
		GlobalPath:  global,
		Env: []string{
			"MEW_REGISTRY=https://env.example/",
		},
		IdentityMew: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err := config.Get(eff, "registry")
	if err != nil {
		t.Fatal(err)
	}
	if v.Raw != "https://env.example/" || v.Source != config.SourceEnv {
		t.Fatalf("registry=%v source=%s", v.Raw, v.Source)
	}
	linker, err := config.Get(eff, "install.linker")
	if err != nil {
		t.Fatal(err)
	}
	if linker.Raw != "hoisted" || linker.Source != config.SourceGlobal {
		t.Fatalf("linker=%v source=%s", linker.Raw, linker.Source)
	}
}

func TestCLIOverridesEnv(t *testing.T) {
	home := testkit.TempHome(t)
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         home,
		ProjectRoot: home,
		GlobalPath:  filepath.Join(home, "missing.jsonc"),
		Env:         []string{"MEW_OFFLINE=false"},
		CLI:         map[string]any{"offline": true},
		IdentityMew: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	v, _ := config.Get(eff, "offline")
	if v.Raw != true || v.Source != config.SourceCLI {
		t.Fatalf("got %+v", v)
	}
}

func TestMewIgnoresNpmrc(t *testing.T) {
	root := testkit.ModuleRoot(t)
	proj := filepath.Join(root, "fixtures", "identity", "mew-native")
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         proj,
		ProjectRoot: proj,
		GlobalPath:  filepath.Join(proj, "no-global.jsonc"),
		IdentityMew: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err := config.Get(eff, "registry")
	if err != nil {
		t.Fatal(err)
	}
	if v.Raw != "https://example.com/mew-registry" {
		t.Fatalf("registry should come from m.jsonc, got %v", v.Raw)
	}
}

func TestRejectRawToken(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	err := config.SetFile(path, "registry.auth_token_env", "npm_0123456789abcdef0123")
	if err == nil {
		t.Fatal("expected reject secret-like value")
	}
	err = config.SetFile(path, "registry.auth_token_env", "NPM_TOKEN")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetRefusesComments(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	if err := os.WriteFile(path, []byte("{\n  // comment\n  \"registry\": \"https://a.example/\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := config.SetFile(path, "offline", true)
	if err == nil {
		t.Fatal("expected comment refuse")
	}
}

func TestMalformedFailClosed(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	_ = os.WriteFile(path, []byte("{not json"), 0o644)
	_, err := config.Load(context.Background(), config.LoadOptions{
		CWD: home, ProjectRoot: home, ProjectPath: path,
		GlobalPath: filepath.Join(home, "nope.jsonc"),
	})
	if err == nil {
		t.Fatal("expected malformed error")
	}
}

func TestListSorted(t *testing.T) {
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD: t.TempDir(), ProjectRoot: t.TempDir(),
		GlobalPath: filepath.Join(t.TempDir(), "x"),
	})
	if err != nil {
		t.Fatal(err)
	}
	list := config.List(eff)
	if len(list) == 0 {
		t.Fatal("empty list")
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].Key >= list[i].Key {
			t.Fatalf("unsorted: %s then %s", list[i-1].Key, list[i].Key)
		}
	}
}
