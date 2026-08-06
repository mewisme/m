package dotenv

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	input := `KEY=value
OTHER=123
EMPTY=
`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value" {
		t.Errorf("KEY = %q, want %q", result["KEY"], "value")
	}
	if result["OTHER"] != "123" {
		t.Errorf("OTHER = %q, want %q", result["OTHER"], "123")
	}
	if result["EMPTY"] != "" {
		t.Errorf("EMPTY = %q, want empty", result["EMPTY"])
	}
}

func TestParseExport(t *testing.T) {
	input := `export KEY=value
export OTHER=123
`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value" {
		t.Errorf("KEY = %q, want %q", result["KEY"], "value")
	}
}

func TestParseComments(t *testing.T) {
	input := `# This is a comment
KEY=value # inline comment
# another comment
OTHER=123
`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value" {
		t.Errorf("KEY = %q, want %q", result["KEY"], "value")
	}
}

func TestParseDoubleQuoted(t *testing.T) {
	input := `KEY="value with spaces"
ESCAPED="line1\nline2"
`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value with spaces" {
		t.Errorf("KEY = %q, want %q", result["KEY"], "value with spaces")
	}
	if result["ESCAPED"] != "line1\nline2" {
		t.Errorf("ESCAPED = %q, want %q", result["ESCAPED"], "line1\nline2")
	}
}

func TestParseSingleQuoted(t *testing.T) {
	input := `KEY='value with spaces'
LITERAL='$NOEXPAND ${VAR}'
`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value with spaces" {
		t.Errorf("KEY = %q, want %q", result["KEY"], "value with spaces")
	}
	if result["LITERAL"] != "$NOEXPAND ${VAR}" {
		t.Errorf("LITERAL = %q, want literal", result["LITERAL"])
	}
}

func TestParseVariableExpansion(t *testing.T) {
	input := `HOST=localhost
URL=http://${HOST}:8080
SIMPLE=$HOST
`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result["URL"] != "http://localhost:8080" {
		t.Errorf("URL = %q, want %q", result["URL"], "http://localhost:8080")
	}
	if result["SIMPLE"] != "localhost" {
		t.Errorf("SIMPLE = %q, want %q", result["SIMPLE"], "localhost")
	}
}

func TestParseVariableExpansionDefault(t *testing.T) {
	input := `DEFAULTED=${MISSING:-fallback}
PRESENT=${HOST:-fallback}
HOST=server
`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result["DEFAULTED"] != "fallback" {
		t.Errorf("DEFAULTED = %q, want %q", result["DEFAULTED"], "fallback")
	}
	if result["PRESENT"] != "fallback" {
		t.Errorf("PRESENT = %q, should be empty (HOST not yet defined), got %q", result["PRESENT"], result["PRESENT"])
	}
}

func TestParseBlankLines(t *testing.T) {
	input := `
KEY=value

OTHER=123

`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("got %d keys, want 2", len(result))
	}
}

func TestLoadMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "BASE=base\nSHARED=from_base\n")
	writeFile(t, filepath.Join(dir, ".env.local"), "LOCAL=local\nSHARED=from_local\n")

	env, err := Load([]string{
		filepath.Join(dir, ".env"),
		filepath.Join(dir, ".env.local"),
	})
	if err != nil {
		t.Fatal(err)
	}
	m := environToMap(env)
	if m["BASE"] != "base" {
		t.Errorf("BASE = %q, want base", m["BASE"])
	}
	if m["LOCAL"] != "local" {
		t.Errorf("LOCAL = %q, want local", m["LOCAL"])
	}
	if m["SHARED"] != "from_local" {
		t.Errorf("SHARED = %q, want from_local (later file should override)", m["SHARED"])
	}
}

func TestDiscoverNoMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "")
	writeFile(t, filepath.Join(dir, ".env.local"), "")

	paths := Discover(dir, "")
	if len(paths) != 2 {
		t.Fatalf("got %d paths, want 2: %v", len(paths), paths)
	}
	if filepath.Base(paths[0]) != ".env" {
		t.Errorf("paths[0] = %s, want .env", paths[0])
	}
	if filepath.Base(paths[1]) != ".env.local" {
		t.Errorf("paths[1] = %s, want .env.local", paths[1])
	}
}

func TestDiscoverWithMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "")
	writeFile(t, filepath.Join(dir, ".env.local"), "")
	writeFile(t, filepath.Join(dir, ".env.production"), "")
	writeFile(t, filepath.Join(dir, ".env.production.local"), "")

	paths := Discover(dir, "production")
	if len(paths) != 4 {
		t.Fatalf("got %d paths, want 4: %v", len(paths), paths)
	}
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	expected := []string{".env", ".env.local", ".env.production", ".env.production.local"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("got %v, want %v", names, expected)
	}
}

func TestDiscoverMissingFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env.production"), "")

	paths := Discover(dir, "production")
	if len(paths) != 1 {
		t.Fatalf("got %d paths, want 1: %v", len(paths), paths)
	}
	if filepath.Base(paths[0]) != ".env.production" {
		t.Errorf("got %s, want .env.production", paths[0])
	}
}

func TestDiscoverEmptyDir(t *testing.T) {
	dir := t.TempDir()
	paths := Discover(dir, "development")
	if len(paths) != 0 {
		t.Errorf("got %d paths, want 0", len(paths))
	}
}

func TestMergeEnviron(t *testing.T) {
	// Set up a controlled environment.
	t.Setenv("EXISTING", "keep")
	t.Setenv("OVERRIDE", "old")

	result := MergeEnviron([]string{"OVERRIDE=new", "NEW_VAR=added"})
	m := environToMap(result)
	if m["EXISTING"] != "keep" {
		t.Errorf("EXISTING = %q, want keep", m["EXISTING"])
	}
	if m["OVERRIDE"] != "new" {
		t.Errorf("OVERRIDE = %q, want new", m["OVERRIDE"])
	}
	if m["NEW_VAR"] != "added" {
		t.Errorf("NEW_VAR = %q, want added", m["NEW_VAR"])
	}
}

func TestMergeEnvironEmpty(t *testing.T) {
	result := MergeEnviron(nil)
	if len(result) == 0 {
		t.Error("empty result for nil overlay")
	}
	result = MergeEnviron([]string{})
	if len(result) == 0 {
		t.Error("empty result for empty overlay")
	}
}

func TestLoadMissingFile(t *testing.T) {
	env, err := Load([]string{"/nonexistent/path/.env"})
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 0 {
		t.Errorf("got %d entries from missing file, want 0", len(env))
	}
}

func TestParseEscapedDollar(t *testing.T) {
	input := `LITERAL=\$HOME
`
	result, err := Parse(strings.NewReader(input), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result["LITERAL"] != "$HOME" {
		t.Errorf("LITERAL = %q, want $HOME", result["LITERAL"])
	}
}

// Helpers

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func environToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				m[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	return m
}
