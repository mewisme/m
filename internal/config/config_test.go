package config_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestSetPreservesComments(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	src := "{\n  // keep me\n  \"registry\": \"https://a.example/\"\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.SetFile(path, "offline", true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "// keep me") {
		t.Fatalf("comment lost:\n%s", got)
	}
	if !strings.Contains(got, `"offline": true`) {
		t.Fatalf("value not written:\n%s", got)
	}
	if !strings.Contains(got, `"registry": "https://a.example/"`) {
		t.Fatalf("sibling lost:\n%s", got)
	}
}

func TestSetPreservesCommentsOnExistingKey(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	src := "{\n  /* block */\n  \"offline\": false, // trailing\n  \"registry\": \"https://a.example/\"\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.SetFile(path, "offline", true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{"/* block */", "// trailing", `"offline": true`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "false") {
		t.Fatalf("old value remains:\n%s", got)
	}
}

func TestSetFileDurablePublish(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "nested", "config.jsonc")
	if err := config.SetFile(path, "offline", true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"offline": true`) {
		t.Fatalf("contents:\n%s", b)
	}
	// Second write replaces atomically.
	if err := config.SetFile(path, "prefer-offline", true); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"prefer_offline": true`) || !strings.Contains(string(b), `"offline": true`) {
		t.Fatalf("contents after second set:\n%s", b)
	}
}

func TestSetFileInvalidJSONCNotOverwritten(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	orig := []byte("{not json")
	if err := os.WriteFile(path, orig, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.SetFile(path, "offline", true); err == nil {
		t.Fatal("expected parse failure")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(orig) {
		t.Fatalf("file mutated on failure: %q", got)
	}
}

func TestUnsetFileSiblingsAndIdempotent(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	if err := config.SetFile(path, "install.linker", "hoisted"); err != nil {
		t.Fatal(err)
	}
	if err := config.SetFile(path, "offline", true); err != nil {
		t.Fatal(err)
	}
	if err := config.UnsetFile(path, "install.linker"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "linker") || strings.Contains(string(b), `"install"`) {
		t.Fatalf("empty install object should be dropped:\n%s", b)
	}
	if !strings.Contains(string(b), `"offline": true`) {
		t.Fatalf("sibling offline lost:\n%s", b)
	}
	// Idempotent missing key.
	if err := config.UnsetFile(path, "install.linker"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(b) {
		t.Fatalf("idempotent unset rewrote file")
	}
}

func TestUnsetFileMissingFileIdempotent(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "missing.jsonc")
	if err := config.UnsetFile(path, "offline"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("should not create file: %v", err)
	}
}

func TestUnsetPreservesComments(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	src := "{\n  // keep me\n  \"offline\": true,\n  \"registry\": \"https://a.example/\"\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.UnsetFile(path, "offline"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "// keep me") {
		t.Fatalf("comment lost:\n%s", got)
	}
	if strings.Contains(got, "offline") {
		t.Fatalf("key not removed:\n%s", got)
	}
	if !strings.Contains(got, `"registry": "https://a.example/"`) {
		t.Fatalf("sibling lost:\n%s", got)
	}
	if _, err := config.ParseJSONC(b); err != nil {
		t.Fatalf("result does not parse: %v\n%s", err, got)
	}
}

// Removing the last member of a commented file must still leave valid JSON.
func TestUnsetLastMemberWithComments(t *testing.T) {
	home := testkit.TempHome(t)
	path := filepath.Join(home, "m.jsonc")
	src := "{\n  \"registry\": \"https://a.example/\",\n  // about offline\n  \"offline\": true\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.UnsetFile(path, "offline"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.ParseJSONC(b); err != nil {
		t.Fatalf("result does not parse: %v\n%s", err, b)
	}
	if strings.Contains(string(b), "offline") && !strings.Contains(string(b), "// about offline") {
		t.Fatalf("key not removed:\n%s", b)
	}
	if !strings.Contains(string(b), `"registry"`) {
		t.Fatalf("sibling lost:\n%s", b)
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

func TestAllowedValuesKnownKeys(t *testing.T) {
	if got := config.AllowedValues("log.level"); got != "error|warn|info|debug" {
		t.Fatalf("log.level values=%q", got)
	}
	if got := config.AllowedValues("install.linker"); got != "auto|hoisted|isolated" {
		t.Fatalf("install.linker values=%q", got)
	}
	if got := config.AllowedValues("offline"); got != "true|false" {
		t.Fatalf("offline values=%q", got)
	}
	if got := config.AllowedValues("registry"); got != "" {
		t.Fatalf("registry values=%q want empty", got)
	}
}

func TestListIncludesValuesColumn(t *testing.T) {
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD: t.TempDir(), ProjectRoot: t.TempDir(),
		GlobalPath: filepath.Join(t.TempDir(), "x"),
		Env:        []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var logLevel, registry, linker config.Entry
	var foundLog, foundRegistry, foundLinker bool
	for _, e := range config.List(eff) {
		switch e.Key {
		case "log.level":
			logLevel, foundLog = e, true
		case "registry":
			registry, foundRegistry = e, true
		case "install.linker":
			linker, foundLinker = e, true
		}
	}
	if !foundLog || logLevel.Values != "error|warn|info|debug" {
		t.Fatalf("log.level entry=%+v found=%v", logLevel, foundLog)
	}
	if !foundRegistry || registry.Values != "" {
		t.Fatalf("registry entry=%+v found=%v", registry, foundRegistry)
	}
	if !foundLinker || linker.Values != "auto|hoisted|isolated" {
		t.Fatalf("install.linker entry=%+v found=%v", linker, foundLinker)
	}
}
