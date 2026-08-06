package transform

import (
	"context"
	"strings"
	"testing"
)

func TestEsbuildEngine_Identity(t *testing.T) {
	e := NewEsbuildEngine()
	id := e.Identity()
	if id.Name != "esbuild" {
		t.Fatalf("engine name=%q, want esbuild", id.Name)
	}
	if id.Version == "" {
		t.Fatal("empty engine version")
	}
}

func TestEsbuildEngine_BasicTS(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "test-1",
		SourcePath:      "test.ts",
		SourceBytes:     []byte("const x: number = 1;\nconsole.log(x);\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if len(res.Code) == 0 {
		t.Fatal("empty output code")
	}
	code := string(res.Code)
	if !strings.Contains(code, "console.log") {
		t.Fatalf("unexpected output: %s", code)
	}
	if strings.Contains(code, ": number") {
		t.Fatal("type annotation not stripped")
	}
	if res.CacheStatus != CacheStatusBypass {
		t.Fatalf("cache status %d, want bypass", res.CacheStatus)
	}
	if res.Transformer.Name != "esbuild" {
		t.Fatalf("transformer=%q", res.Transformer.Name)
	}
}

func TestEsbuildEngine_MTS(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "test-mts",
		SourcePath:      "lib.mts",
		SourceBytes:     []byte("export const foo: string = 'bar';\n"),
		SourceDigest:    "fake",
		Loader:          LoaderMTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "export") {
		t.Fatalf("unexpected output: %s", code)
	}
}

func TestEsbuildEngine_CTS(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "test-cts",
		SourcePath:      "lib.cts",
		SourceBytes:     []byte("const x: number = 1;\nmodule.exports = { x };\n"),
		SourceDigest:    "fake",
		Loader:          LoaderCTS,
		Format:          FormatCJS,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "module.exports") {
		t.Fatalf("unexpected output: %s", code)
	}
}

func TestEsbuildEngine_SourceMapInline(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "test-map",
		SourcePath:      "app.ts",
		SourceBytes:     []byte("const x: number = 1;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapInline,
		TargetNodeMajor: 20,
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(res.Code), "sourceMappingURL") {
		t.Fatal("expected source map URL in output")
	}
}

func TestEsbuildEngine_SyntaxError(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "test-err",
		SourcePath:      "bad.ts",
		SourceBytes:     []byte("const x: number = ;\n"), // syntax error
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	_, err := e.Transform(ctx, req)
	if err == nil {
		t.Fatal("expected error for syntax error, got nil")
	}
}

func TestEsbuildEngine_DeterministicOutput(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	src := []byte("const x: number = 1;\nexport default x;\n")
	req := TransformRequest{
		RequestID:       "det-1",
		SourcePath:      "mod.ts",
		SourceBytes:     src,
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	res1, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("first transform: %v", err)
	}
	res2, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("second transform: %v", err)
	}
	if string(res1.Code) != string(res2.Code) {
		t.Fatal("non-deterministic output")
	}
	if res1.OutputDigest != res2.OutputDigest {
		t.Fatal("non-deterministic digest")
	}
}

func TestEsbuildEngine_ContextCancellation(t *testing.T) {
	e := NewEsbuildEngine()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := TransformRequest{
		RequestID:       "cancel-1",
		SourcePath:      "app.ts",
		SourceBytes:     []byte("const x = 1;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	_, err := e.Transform(ctx, req)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestEsbuildEngine_AsyncFunctions(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "async-1",
		SourcePath:      "async.ts",
		SourceBytes:     []byte("export async function fetch(): Promise<string> { return 'ok'; }\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "async") {
		t.Fatalf("async keyword lost: %s", code)
	}
}

func TestEsbuildEngine_DynamicImport(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "dyn-1",
		SourcePath:      "dyn.ts",
		SourceBytes:     []byte("const m = await import('./mod');\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "import(") {
		t.Fatalf("dynamic import lost: %s", code)
	}
}

// ── JSX tests ──────────────────────────────────────────────────────

func TestEsbuildEngine_JSXClassicRuntime(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "jsx-classic",
		SourcePath:      "app.tsx",
		SourceBytes:     []byte("const el = <div>hello</div>;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTSX,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts:  NormalizedOptions{JSX: "react"},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "React.createElement") {
		t.Fatalf("classic JSX not emitted: %s", code)
	}
}

func TestEsbuildEngine_JSXAutomaticRuntime(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "jsx-automatic",
		SourcePath:      "app.tsx",
		SourceBytes:     []byte("const el = <div>hello</div>;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTSX,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts:  NormalizedOptions{JSX: "react-jsx"},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "jsx") || !strings.Contains(code, "react/jsx-runtime") {
		t.Fatalf("automatic JSX not emitted: %s", code)
	}
}

func TestEsbuildEngine_JSXPreserve(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "jsx-preserve",
		SourcePath:      "app.tsx",
		SourceBytes:     []byte("const el = <div>hello</div>;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTSX,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts:  NormalizedOptions{JSX: "preserve"},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "<div") {
		t.Fatalf("JSX not preserved: %s", code)
	}
}

func TestEsbuildEngine_JSXCustomFactory(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "jsx-custom-factory",
		SourcePath:      "app.tsx",
		SourceBytes:     []byte("const el = <div>hello</div>;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTSX,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{
			JSX:        "react",
			JSXFactory: "h",
		},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "h(") {
		t.Fatalf("custom JSX factory not used: %s", code)
	}
}

func TestEsbuildEngine_JSXCustomImportSource(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "jsx-preact",
		SourcePath:      "app.tsx",
		SourceBytes:     []byte("const el = <div>hello</div>;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTSX,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{
			JSX:             "react-jsx",
			JSXImportSource: "preact",
		},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "preact/jsx-runtime") {
		t.Fatalf("custom import source not used: %s", code)
	}
}

// ── TSX loader tests ───────────────────────────────────────────────

func TestEsbuildEngine_TSXLoader(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "tsx-loader",
		SourcePath:      "component.tsx",
		SourceBytes:     []byte("const x: number = 1;\nconst el = <div>{x}</div>;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTSX,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts:  NormalizedOptions{JSX: "react-jsx"},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if strings.Contains(code, ": number") {
		t.Fatal("type annotation not stripped in TSX")
	}
	if !strings.Contains(code, "jsx") {
		t.Fatalf("JSX not transformed in TSX: %s", code)
	}
}

// ── Decorator tests ────────────────────────────────────────────────

func TestEsbuildEngine_DecoratorLegacy(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	src := "function sealed(target: any) {}\n@sealed\nclass MyClass {}"
	req := TransformRequest{
		RequestID:       "decorator-legacy",
		SourcePath:      "decorator.ts",
		SourceBytes:     []byte(src),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{
			ExperimentalDecorators: true,
		},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "__decorate") {
		t.Fatalf("legacy decorator helper not emitted: %s", code)
	}
}

func TestEsbuildEngine_DecoratorMetadata(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	src := "function log(target: any, key: string) {}\nclass Example {\n  @log\n  method() {}\n}"
	req := TransformRequest{
		RequestID:       "decorator-meta",
		SourcePath:      "decorator.ts",
		SourceBytes:     []byte(src),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{
			ExperimentalDecorators: true,
			EmitDecoratorMetadata:  true,
		},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "__decorate") {
		t.Fatalf("decorator helper not emitted: %s", code)
	}
}

func TestEsbuildEngine_DecoratorWorksWithoutFlag(t *testing.T) {
	// esbuild always accepts decorators; the experimentalDecorators tsconfig
	// flag gates the TypeScript type-checker, not the transpiler.
	e := NewEsbuildEngine()
	ctx := context.Background()

	src := "function sealed(target: any) {}\n@sealed\nclass MyClass {}"
	req := TransformRequest{
		RequestID:       "decorator-no-flag",
		SourcePath:      "decorator.ts",
		SourceBytes:     []byte(src),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	code := string(res.Code)
	if !strings.Contains(code, "__decorate") {
		t.Fatalf("decorator helper not emitted: %s", code)
	}
}

// ── Source map tests ───────────────────────────────────────────────

func TestEsbuildEngine_SourceMapExternal(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "sm-external",
		SourcePath:      "app.ts",
		SourceBytes:     []byte("const x: number = 1;\nexport default x;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapExternal,
		TargetNodeMajor: 20,
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if len(res.SourceMap) == 0 {
		t.Fatal("expected external source map bytes")
	}
	if strings.Contains(string(res.Code), "sourceMappingURL") {
		t.Fatal("external source map must not be inlined")
	}
}

func TestEsbuildEngine_SourceMapFromTsconfig(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "sm-tsconfig",
		SourcePath:      "app.ts",
		SourceBytes:     []byte("const x: number = 1;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{
			SourceMap: true,
		},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if len(res.SourceMap) == 0 {
		t.Fatal("expected source map from tsconfig sourceMap:true")
	}
}

func TestEsbuildEngine_InlineSourceMapOverride(t *testing.T) {
	e := NewEsbuildEngine()
	ctx := context.Background()

	req := TransformRequest{
		RequestID:       "sm-inline-override",
		SourcePath:      "app.ts",
		SourceBytes:     []byte("const x: number = 1;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{
			InlineSourceMap: true,
		},
	}

	res, err := e.Transform(ctx, req)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(res.Code), "sourceMappingURL") {
		t.Fatal("expected inline source map from inlineSourceMap:true")
	}
}

// ── Cache key includes new options ────────────────────────────────

func TestCacheKeyVariesByJSXOptions(t *testing.T) {
	src := []byte("const el = <div/>;")
	req1 := TransformRequest{
		RequestID: "ck-1", SourcePath: "a.tsx", SourceBytes: src,
		SourceDigest: "fake", Loader: LoaderTSX, Format: FormatESM,
		SourceMapMode: SourceMapNone, TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{JSX: "react"},
	}
	req2 := TransformRequest{
		RequestID: "ck-2", SourcePath: "a.tsx", SourceBytes: src,
		SourceDigest: "fake", Loader: LoaderTSX, Format: FormatESM,
		SourceMapMode: SourceMapNone, TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{JSX: "react-jsx"},
	}
	id := EngineIdentity{Name: "esbuild", Version: "1.0"}
	k1 := CacheKey(req1, id)
	k2 := CacheKey(req2, id)
	if k1 == k2 {
		t.Fatal("cache keys must differ when JSX mode differs")
	}
}

func TestCacheKeyVariesByDecoratorOptions(t *testing.T) {
	src := []byte("const x = 1;")
	req1 := TransformRequest{
		RequestID: "ck-3", SourcePath: "a.ts", SourceBytes: src,
		SourceDigest: "fake", Loader: LoaderTS, Format: FormatESM,
		SourceMapMode: SourceMapNone, TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{ExperimentalDecorators: true},
	}
	req2 := TransformRequest{
		RequestID: "ck-4", SourcePath: "a.ts", SourceBytes: src,
		SourceDigest: "fake", Loader: LoaderTS, Format: FormatESM,
		SourceMapMode: SourceMapNone, TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{ExperimentalDecorators: true, EmitDecoratorMetadata: true},
	}
	id := EngineIdentity{Name: "esbuild", Version: "1.0"}
	k1 := CacheKey(req1, id)
	k2 := CacheKey(req2, id)
	if k1 == k2 {
		t.Fatal("cache keys must differ when decorator metadata differs")
	}
}

func TestCacheKeyVariesByImportSource(t *testing.T) {
	src := []byte("const el = <div/>;")
	req1 := TransformRequest{
		RequestID: "ck-5", SourcePath: "a.tsx", SourceBytes: src,
		SourceDigest: "fake", Loader: LoaderTSX, Format: FormatESM,
		SourceMapMode: SourceMapNone, TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{JSX: "react-jsx", JSXImportSource: "react"},
	}
	req2 := TransformRequest{
		RequestID: "ck-6", SourcePath: "a.tsx", SourceBytes: src,
		SourceDigest: "fake", Loader: LoaderTSX, Format: FormatESM,
		SourceMapMode: SourceMapNone, TargetNodeMajor: 20,
		NormalizedOpts: NormalizedOptions{JSX: "react-jsx", JSXImportSource: "preact"},
	}
	id := EngineIdentity{Name: "esbuild", Version: "1.0"}
	k1 := CacheKey(req1, id)
	k2 := CacheKey(req2, id)
	if k1 == k2 {
		t.Fatal("cache keys must differ when JSX import source differs")
	}
}
