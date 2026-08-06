// Package runtime_test provides runtime stabilization conformance tests.
package runtime_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/mewisme/mew/internal/transform"
)

// TestTransformCacheRoundTrip verifies a transform → write → read → verify cycle
// produces identical output and no corruption.
func TestTransformCacheRoundTrip(t *testing.T) {
	engine := transform.NewEsbuildEngine()
	identity := engine.Identity()
	src := `const msg: string = "conformance"; export default msg;`
	req := transform.TransformRequest{
		SourcePath:    "conformance.ts",
		SourceBytes:   []byte(src),
		Loader:        transform.LoaderTS,
		Format:        transform.FormatESM,
		SourceMapMode: transform.SourceMapExternal,
	}

	ctx := context.Background()
	result, err := engine.Transform(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	key := transform.CacheKey(req, identity)
	dir := t.TempDir()

	if err := transform.WriteCache(dir, key, &result); err != nil {
		t.Fatal(err)
	}

	readBack, err := transform.TryReadCache(dir, key)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(readBack.Code, result.Code) {
		t.Fatalf("code mismatch: wrote %d bytes, read %d bytes", len(result.Code), len(readBack.Code))
	}
	if !bytes.Equal(readBack.SourceMap, result.SourceMap) {
		t.Fatal("source map mismatch after round-trip")
	}
}

// TestCacheKeyStability verifies cache keys are deterministic across calls.
func TestCacheKeyStability(t *testing.T) {
	engine := transform.NewEsbuildEngine()
	identity := engine.Identity()
	src := []byte(`const x: number = 1; export {x};`)
	req := transform.TransformRequest{
		SourcePath:    "stable.ts",
		SourceBytes:   src,
		Loader:        transform.LoaderTS,
		Format:        transform.FormatESM,
		SourceMapMode: transform.SourceMapNone,
	}

	key1 := transform.CacheKey(req, identity)
	key2 := transform.CacheKey(req, identity)

	if key1 != key2 {
		t.Fatalf("cache key not stable: %q != %q", key1, key2)
	}

	// Different source must produce different key.
	req2 := req
	req2.SourceBytes = []byte(`const x: number = 2; export {x};`)
	key3 := transform.CacheKey(req2, identity)

	if key1 == key3 {
		t.Fatal("different sources produced same cache key")
	}
}

// TestCacheSchemaVersion verifies the cache schema version is the expected value.
func TestCacheSchemaVersion(t *testing.T) {
	if transform.CacheSchemaVersion != 1 {
		t.Fatalf("unexpected cache schema version: %d", transform.CacheSchemaVersion)
	}
}

// TestSourceMapRoundTrip verifies source maps survive the cache round-trip.
func TestSourceMapRoundTrip(t *testing.T) {
	engine := transform.NewEsbuildEngine()
	identity := engine.Identity()

	cases := []struct {
		name string
		mode transform.SourceMapMode
		src  string
	}{
		{"none", transform.SourceMapNone, `const x = 1;`},
		{"external", transform.SourceMapExternal, `const x: number = 1;`},
		{"inline", transform.SourceMapInline, `const y: string = "x";`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := transform.TransformRequest{
				SourcePath:    "test.ts",
				SourceBytes:   []byte(c.src),
				Loader:        transform.LoaderTS,
				Format:        transform.FormatESM,
				SourceMapMode: c.mode,
			}

			ctx := context.Background()
			result, err := engine.Transform(ctx, req)
			if err != nil {
				t.Fatal(err)
			}

			key := transform.CacheKey(req, identity)
			dir := t.TempDir()

			if err := transform.WriteCache(dir, key, &result); err != nil {
				t.Fatal(err)
			}

			readBack, err := transform.TryReadCache(dir, key)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(readBack.Code, result.Code) {
				t.Fatalf("code mismatch: %s", c.name)
			}
			if !bytes.Equal(readBack.SourceMap, result.SourceMap) {
				t.Fatalf("source map mismatch: %s", c.name)
			}
			if readBack.OutputDigest != result.OutputDigest {
				t.Fatalf("output digest mismatch: %s", c.name)
			}
		})
	}
}
