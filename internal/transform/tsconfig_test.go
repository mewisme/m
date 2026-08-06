package transform

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	opts, err := NormalizeOptions(chain)
	if err != nil {
		t.Fatal(err)
	}
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
	opts, err := NormalizeOptions(chain)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Target != "ES2020" {
		t.Fatalf("target=%s, want ES2020 (parent value when child absent)", opts.Target)
	}
	if opts.Module != "ESNext" {
		t.Fatalf("module=%s, want ESNext (child override)", opts.Module)
	}
}

func TestNormalizeOptionsEmptyChain(t *testing.T) {
	opts, err := NormalizeOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
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
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrExtendsCycle {
		t.Fatalf("expected ConfigErrExtendsCycle, got %s", cfgErr.Kind)
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
	d2 := TsconfigChainDigest(chain)
	if d1 != d2 {
		t.Fatalf("same chain different digests: %s vs %s", d1, d2)
	}
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

// --- New fail-closed error tests ---

func TestMalformedJSONC(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{invalid`)

	_, err := LoadTsconfigChain(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrParse {
		t.Fatalf("expected ConfigErrParse, got %s", cfgErr.Kind)
	}
	if cfgErr.Path != path {
		t.Fatalf("path=%s, want %s", cfgErr.Path, path)
	}
}

func TestUnreadableConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tsconfig.json")
	// Create a directory with the same name so os.ReadFile fails.
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTsconfigChain(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrIO {
		t.Fatalf("expected ConfigErrIO, got %s", cfgErr.Kind)
	}
}

func TestConfigPathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	// Create tsconfig.json as a directory.
	cfgDir := filepath.Join(sub, "tsconfig.json")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := DiscoverTsconfig(sub)
	if err == nil {
		t.Fatal("expected error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrIO {
		t.Fatalf("expected ConfigErrIO, got %s", cfgErr.Kind)
	}
}

func TestMissingRelativeExtendsFile(t *testing.T) {
	dir := t.TempDir()
	child := `{"extends":"./nonexistent.json"}`
	path := writeJSONC(t, dir, "tsconfig.json", child)

	_, err := LoadTsconfigChain(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrExtendsMissing {
		t.Fatalf("expected ConfigErrExtendsMissing, got %s", cfgErr.Kind)
	}
}

func TestExtendsDepthOverflow(t *testing.T) {
	dir := t.TempDir()
	// Create a chain that exceeds maxTsconfigDepth (20).
	// base0 is the root, and each level extends the previous.
	writeJSONC(t, dir, "base0.json", `{"compilerOptions":{}}`)
	prev := "base0.json"
	for i := 1; i <= maxTsconfigDepth+1; i++ {
		name := "cfg" + strings.Repeat("x", i) + ".json"
		writeJSONC(t, dir, name, `{"extends":"./`+prev+`"}`)
		prev = name
	}

	_, err := LoadTsconfigChain(filepath.Join(dir, prev))
	if err == nil {
		t.Fatal("expected depth overflow error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrExtendsDepth {
		t.Fatalf("expected ConfigErrExtendsDepth, got %s", cfgErr.Kind)
	}
}

func TestNonStringExtends(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"extends":42}`)

	_, err := LoadTsconfigChain(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrExtendsInvalid {
		t.Fatalf("expected ConfigErrExtendsInvalid, got %s", cfgErr.Kind)
	}
}

func TestEmptyExtends(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"extends":""}`)

	_, err := LoadTsconfigChain(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrExtendsInvalid {
		t.Fatalf("expected ConfigErrExtendsInvalid, got %s", cfgErr.Kind)
	}
}

func TestUnsupportedPackageExtends(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"extends":"@scope/tsconfig"}`)

	_, err := LoadTsconfigChain(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrExtendsPackage {
		t.Fatalf("expected ConfigErrExtendsPackage, got %s", cfgErr.Kind)
	}
}

func TestInvalidCompilerOptionShape(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"compilerOptions":{"target":42}}`)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NormalizeOptions(chain)
	if err == nil {
		t.Fatal("expected option error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrOptionInvalid {
		t.Fatalf("expected ConfigErrOptionInvalid, got %s", cfgErr.Kind)
	}
}

func TestCompilerOptionsNotAnObject(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"compilerOptions":"strict"}`)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NormalizeOptions(chain)
	if err == nil {
		t.Fatal("expected option error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrOptionInvalid {
		t.Fatalf("expected ConfigErrOptionInvalid, got %s", cfgErr.Kind)
	}
}

func TestInvalidBooleanOption(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"compilerOptions":{"useDefineForClassFields":"yes"}}`)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NormalizeOptions(chain)
	if err == nil {
		t.Fatal("expected option error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrOptionInvalid {
		t.Fatalf("expected ConfigErrOptionInvalid, got %s", cfgErr.Kind)
	}
}

func TestInvalidPathsOption(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"compilerOptions":{"paths":"bad"}}`)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NormalizeOptions(chain)
	if err == nil {
		t.Fatal("expected option error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrOptionInvalid {
		t.Fatalf("expected ConfigErrOptionInvalid, got %s", cfgErr.Kind)
	}
}

func TestConfigErrorPathPreserved(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{invalid`)

	_, err := LoadTsconfigChain(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Path != path {
		t.Fatalf("path=%s, want %s", cfgErr.Path, path)
	}
	// Error message must contain the path but not expose raw file contents.
	// The json parse error may reference character position but not the file bytes.
	if !strings.Contains(cfgErr.Error(), path) {
		t.Fatalf("error does not contain path: %s", cfgErr.Error())
	}
}

func TestNormalizeJSXOptions(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"compilerOptions":{"jsx":"react-jsx","jsxFactory":"h","jsxFragmentFactory":"Fragment","jsxImportSource":"preact"}}`
	path := writeJSONC(t, dir, "tsconfig.json", cfg)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	opts, err := NormalizeOptions(chain)
	if err != nil {
		t.Fatal(err)
	}
	if opts.JSX != "react-jsx" {
		t.Fatalf("jsx=%s, want react-jsx", opts.JSX)
	}
	if opts.JSXFactory != "h" {
		t.Fatalf("jsxFactory=%s, want h", opts.JSXFactory)
	}
	if opts.JSXFragmentFactory != "Fragment" {
		t.Fatalf("jsxFragmentFactory=%s, want Fragment", opts.JSXFragmentFactory)
	}
	if opts.JSXImportSource != "preact" {
		t.Fatalf("jsxImportSource=%s, want preact", opts.JSXImportSource)
	}
}

func TestNormalizeDecoratorOptions(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"compilerOptions":{"experimentalDecorators":true,"emitDecoratorMetadata":true}}`
	path := writeJSONC(t, dir, "tsconfig.json", cfg)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	opts, err := NormalizeOptions(chain)
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ExperimentalDecorators {
		t.Fatal("experimentalDecorators should be true")
	}
	if !opts.EmitDecoratorMetadata {
		t.Fatal("emitDecoratorMetadata should be true")
	}
}

func TestNormalizeSourceMapOptions(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"compilerOptions":{"sourceMap":true,"inlineSourceMap":true,"inlineSources":true,"sourceRoot":"/src","mapRoot":"/maps"}}`
	path := writeJSONC(t, dir, "tsconfig.json", cfg)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	opts, err := NormalizeOptions(chain)
	if err != nil {
		t.Fatal(err)
	}
	if !opts.SourceMap {
		t.Fatal("sourceMap should be true")
	}
	if !opts.InlineSourceMap {
		t.Fatal("inlineSourceMap should be true")
	}
	if !opts.InlineSources {
		t.Fatal("inlineSources should be true")
	}
	if opts.SourceRoot != "/src" {
		t.Fatalf("sourceRoot=%s, want /src", opts.SourceRoot)
	}
	if opts.MapRoot != "/maps" {
		t.Fatalf("mapRoot=%s, want /maps", opts.MapRoot)
	}
}

func TestInvalidJSXFactoryType(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"compilerOptions":{"jsxFactory":42}}`)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NormalizeOptions(chain)
	if err == nil {
		t.Fatal("expected option error for non-string jsxFactory")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrOptionInvalid {
		t.Fatalf("expected ConfigErrOptionInvalid, got %s", cfgErr.Kind)
	}
}

func TestInvalidExperimentalDecoratorsType(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONC(t, dir, "tsconfig.json", `{"compilerOptions":{"experimentalDecorators":"yes"}}`)

	chain, err := LoadTsconfigChain(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NormalizeOptions(chain)
	if err == nil {
		t.Fatal("expected option error for non-bool experimentalDecorators")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Kind != ConfigErrOptionInvalid {
		t.Fatalf("expected ConfigErrOptionInvalid, got %s", cfgErr.Kind)
	}
}

func TestNormalizeOptionsDigestIncludesNewFields(t *testing.T) {
	opts1 := NormalizedOptions{JSX: "react"}
	opts2 := NormalizedOptions{JSX: "react-jsx"}
	if opts1.Digest() == opts2.Digest() {
		t.Fatal("digests must differ when JSX mode differs")
	}

	opts3 := NormalizedOptions{ExperimentalDecorators: true}
	opts4 := NormalizedOptions{ExperimentalDecorators: true, EmitDecoratorMetadata: true}
	if opts3.Digest() == opts4.Digest() {
		t.Fatal("digests must differ when decorator metadata differs")
	}
}
