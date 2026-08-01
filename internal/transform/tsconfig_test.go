package transform

import (
	"encoding/json"
	"strings"
	"os"
	"path/filepath"
	"testing"
)

func writeJSONC(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStripJSONCComments(t *testing.T) {
	input := `{
		// single-line comment
		"compilerOptions": {
			"target": "ES2022", /* block comment */
			"strict": true
		}
	}`
	cleaned := stripJSONC([]byte(input))
	var m map[string]any
	if err := json.Unmarshal(cleaned, &m); err != nil {
		t.Fatalf("JSONC parse failed: %v\ncleaned:\n%s", err, string(cleaned))
	}
	co := m["compilerOptions"].(map[string]any)
	if co["target"] != "ES2022" {
		t.Fatalf("target=%v", co["target"])
	}
	if co["strict"] != true {
		t.Fatalf("strict=%v", co["strict"])
	}
}

func TestStripJSONCTrailingCommas(t *testing.T) {
	input := `{
		"compilerOptions": {
			"target": "ES2022",
			"module": "ESNext",
		},
	}`
	cleaned := stripJSONC([]byte(input))
	var m map[string]any
	if err := json.Unmarshal(cleaned, &m); err != nil {
		t.Fatalf("JSONC trailing comma parse failed: %v\ncleaned:\n%s", err, string(cleaned))
	}
}

func TestLoadTsconfigChainSingle(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"compilerOptions":{"target":"ES2022","strict":true}}`
	path := writeJSONC(t, dir, "tsconfig.json", cfg)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 1 {
		t.Fatalf("chain len=%d, want 1", len(chain))
	}
	if chain[0].Digest == "" {
		t.Fatal("empty digest")
	}
}

func TestLoadTsconfigChainExtends(t *testing.T) {
	dir := t.TempDir()
	base := `{"compilerOptions":{"target":"ES2020","strict":true}}`
	writeJSONC(t, dir, "base.json", base)
	child := `{"extends":"./base.json","compilerOptions":{"target":"ES2022"}}`
	path := writeJSONC(t, dir, "tsconfig.json", child)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 2 {
		t.Fatalf("chain len=%d, want 2", len(chain))
	}
	// First element is parent, second is child.
	if chain[0].Path != filepath.Join(dir, "base.json") {
		t.Fatalf("parent path=%s", chain[0].Path)
	}
	if chain[1].Path != filepath.Join(dir, "tsconfig.json") {
		t.Fatalf("child path=%s", chain[1].Path)
	}
}

func TestNormalizeOptionsChildOverridesParent(t *testing.T) {
	dir := t.TempDir()
	base := `{"compilerOptions":{"target":"ES2020","module":"CommonJS","useDefineForClassFields":true}}`
	writeJSONC(t, dir, "base.json", base)
	child := `{"extends":"./base.json","compilerOptions":{"target":"ES2022","module":"ESNext","useDefineForClassFields":false}}`
	path := writeJSONC(t, dir, "tsconfig.json", child)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	opts := NormalizeOptions(chain)
	if opts.Target != "ES2022" {
		t.Fatalf("target=%s, want ES2022 (child should override parent)", opts.Target)
	}
	if opts.Module != "ESNext" {
		t.Fatalf("module=%s, want ESNext (child should override parent)", opts.Module)
	}
	if opts.UseDefineForClassFields != false {
		t.Fatalf("useDefineForClassFields=%v, want false (child explicit false)", opts.UseDefineForClassFields)
	}
}

func TestNormalizeOptionsParentOnlyAppliesWhenChildAbsent(t *testing.T) {
	dir := t.TempDir()
	base := `{"compilerOptions":{"target":"ES2020","strict":true}}`
	writeJSONC(t, dir, "base.json", base)
	child := `{"extends":"./base.json","compilerOptions":{"module":"ESNext"}}`
	path := writeJSONC(t, dir, "tsconfig.json", child)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	opts := NormalizeOptions(chain)
	// target comes from parent (child doesn't override it).
	if opts.Target != "ES2020" {
		t.Fatalf("target=%s, want ES2020 (parent value when child absent)", opts.Target)
	}
	// module comes from child override.
	if opts.Module != "ESNext" {
		t.Fatalf("module=%s, want ESNext (child override)", opts.Module)
	}
}

func TestNormalizeOptionsEmptyChain(t *testing.T) {
	opts := NormalizeOptions(nil)
	if opts.Target != "" {
		t.Fatalf("target=%s, want empty", opts.Target)
	}
}

func TestTsconfigCycleDetection(t *testing.T) {
	dir := t.TempDir()
	a := `{"extends":"./b.json"}`
	writeJSONC(t, dir, "a.json", a)
	b := `{"extends":"./a.json"}`
	writeJSONC(t, dir, "b.json", b)

	_, err := LoadTsconfigChain(filepath.Join(dir, "a.json"))
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if err.Error() == "" || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got: %v", err)
	}
}

func TestTsconfigChainDigest(t *testing.T) {
	chain := []TsconfigFile{
		{Digest: "aaa"},
		{Digest: "bbb"},
	}
	d1 := TsconfigChainDigest(chain)
	if d1 == "" {
		t.Fatal("empty digest")
	}
	// Same digests → same chain digest.
	d2 := TsconfigChainDigest(chain)
	if d1 != d2 {
		t.Fatalf("same chain different digests: %s vs %s", d1, d2)
	}
	// Different child → different chain digest.
	chain2 := []TsconfigFile{
		{Digest: "aaa"},
		{Digest: "ccc"},
	}
	d3 := TsconfigChainDigest(chain2)
	if d1 == d3 {
		t.Fatal("different chains produced same digest")
	}
}

func TestDiscoverTsconfigWalkUp(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "src", "components")
	writeJSONC(t, dir, "tsconfig.json", `{"compilerOptions":{"target":"ES2022"}}`)

	path, err := DiscoverTsconfig(sub)
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("no tsconfig found")
	}
}

func TestDiscoverTsconfigNotFound(t *testing.T) {
	dir := t.TempDir()
	path, err := DiscoverTsconfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Fatalf("found tsconfig at %s, want empty", path)
	}
}


