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
	if res.CacheStatus != CacheStatusMiss {
		t.Fatalf("cache status %d, want miss", res.CacheStatus)
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
