package transform_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/transform"
)

func BenchmarkEngineTransform(b *testing.B) {
	engine := transform.NewEsbuildEngine()
	src := `const x: string = "hello"; console.log(x);`
	req := transform.TransformRequest{
		SourcePath:    "test.ts",
		SourceBytes:   []byte(src),
		Loader:        transform.LoaderTS,
		Format:        transform.FormatESM,
		SourceMapMode: transform.SourceMapNone,
	}
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		_, err := engine.Transform(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheWriteRead(b *testing.B) {
	engine := transform.NewEsbuildEngine()
	identity := engine.Identity()
	src := `const x: string = "bench"; console.log(x);`
	req := transform.TransformRequest{
		SourcePath:    "bench.ts",
		SourceBytes:   []byte(src),
		Loader:        transform.LoaderTS,
		Format:        transform.FormatESM,
		SourceMapMode: transform.SourceMapNone,
	}
	ctx := context.Background()
	result, err := engine.Transform(ctx, req)
	if err != nil {
		b.Fatal(err)
	}
	key := transform.CacheKey(req, identity)
	dir := b.TempDir()

	b.ResetTimer()
	for b.Loop() {
		if err := transform.WriteCache(dir, key, &result); err != nil {
			b.Fatal(err)
		}
		_, err := transform.TryReadCache(dir, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheKeyDeterminism(b *testing.B) {
	engine := transform.NewEsbuildEngine()
	identity := engine.Identity()
	req := transform.TransformRequest{
		SourcePath:    "det.ts",
		SourceBytes:   []byte(`const a: number = 1;`),
		Loader:        transform.LoaderTS,
		Format:        transform.FormatESM,
		SourceMapMode: transform.SourceMapNone,
	}
	for b.Loop() {
		_ = transform.CacheKey(req, identity)
	}
}
