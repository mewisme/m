package transform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
)

func testEngine() Engine { return NewEsbuildEngine() }

func testRequest() TransformRequest {
	return TransformRequest{
		RequestID:       "test",
		SourcePath:      "test.ts",
		SourceBytes:     []byte("const x: number = 1;\n"),
		SourceDigest:    "fake",
		Loader:          LoaderTS,
		Format:          FormatESM,
		SourceMapMode:   SourceMapNone,
		TargetNodeMajor: 20,
	}
}

func TestTryReadCacheEmptyDir(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)
	key := CacheKey(testRequest(), testEngine().Identity())

	result, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatalf("TryReadCache empty: %v", err)
	}
	if result != nil {
		t.Fatal("expected nil for empty cache")
	}
}

func TestWriteAndReadCache(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()
	key := CacheKey(req, engine.Identity())

	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	if err := WriteCache(cacheDir, key, &result); err != nil {
		t.Fatal(err)
	}

	cached, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatalf("TryReadCache: %v", err)
	}
	if cached == nil {
		t.Fatal("expected cache hit")
	}
	if cached.CacheStatus != CacheStatusHit {
		t.Fatalf("cache status=%d, want hit", cached.CacheStatus)
	}
}

func TestCacheColdWarmByteEquivalence(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()

	// Cold: run engine.
	cold, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Write and read back.
	key := CacheKey(req, engine.Identity())
	if err := WriteCache(cacheDir, key, &cold); err != nil {
		t.Fatal(err)
	}

	warm, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatalf("TryReadCache: %v", err)
	}
	if warm == nil {
		t.Fatal("expected cache hit")
	}

	// Byte equivalence.
	if !VerifyCachedResult(warm, &cold) {
		t.Fatal("cold/warm byte mismatch")
	}
	if string(cold.Code) != string(warm.Code) {
		t.Fatal("cold/warm code mismatch")
	}
}

func TestCacheDigestEquivalence(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()

	cold, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	key := CacheKey(req, engine.Identity())
	if err := WriteCache(cacheDir, key, &cold); err != nil {
		t.Fatal(err)
	}

	warm, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatalf("TryReadCache: %v", err)
	}
	if warm == nil {
		t.Fatal("expected cache hit")
	}

	// OutputDigest must match between cold (engine) and warm (cache).
	if cold.OutputDigest != warm.OutputDigest {
		t.Fatalf("digest mismatch: cold=%s warm=%s", cold.OutputDigest, warm.OutputDigest)
	}
}

func TestCacheOutputDigestIndependentOfCache(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()

	// Transform twice; digest must be identical regardless of cache state.
	req := testRequest()
	r1, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if r1.OutputDigest != r2.OutputDigest {
		t.Fatal("non-deterministic engine output digest")
	}

	// Cache r2.
	key := CacheKey(req, engine.Identity())
	if err := WriteCache(cacheDir, key, &r2); err != nil {
		t.Fatal(err)
	}

	cached, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatal(err)
	}
	if cached == nil {
		t.Fatal("expected cache hit")
	}

	// Cache hit digest must match engine digest.
	if r1.OutputDigest != cached.OutputDigest {
		t.Fatalf("cache hit digest %s != engine digest %s", cached.OutputDigest, r1.OutputDigest)
	}
}

func TestTryReadCacheCorruptMeta(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()
	key := CacheKey(req, engine.Identity())

	// Write a valid entry.
	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteCache(cacheDir, key, &result); err != nil {
		t.Fatal(err)
	}

	// Corrupt the metadata.
	metaPath := CacheKeyPath(cacheDir, key) + ".meta"
	if err := os.WriteFile(metaPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	// TryReadCache should return an error (corrupt meta).
	_, err = TryReadCache(cacheDir, key)
	if err == nil {
		t.Fatal("expected error for corrupt meta")
	}

	// Corrupt meta file should be removed.
	if _, statErr := os.Stat(metaPath); !os.IsNotExist(statErr) {
		t.Fatal("corrupt meta was not removed")
	}
}

func TestTryReadCacheCorruptCode(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()
	key := CacheKey(req, engine.Identity())

	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteCache(cacheDir, key, &result); err != nil {
		t.Fatal(err)
	}

	// Corrupt the code file.
	codePath := CacheKeyPath(cacheDir, key) + ".code"
	if err := os.WriteFile(codePath, []byte("corrupted code"), 0o644); err != nil {
		t.Fatal(err)
	}

	// TryReadCache should return an error (digest mismatch).
	_, err = TryReadCache(cacheDir, key)
	if err == nil {
		t.Fatal("expected error for corrupt code")
	}

	// Corrupt code, map, and meta files should be removed.
	for _, ext := range []string{".code", ".map", ".meta"} {
		if _, statErr := os.Stat(CacheKeyPath(cacheDir, key) + ext); !os.IsNotExist(statErr) {
			t.Errorf("corrupt %s was not removed", ext)
		}
	}
}

func TestTryReadCacheMissingCode(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()
	key := CacheKey(req, engine.Identity())

	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteCache(cacheDir, key, &result); err != nil {
		t.Fatal(err)
	}

	// Delete the code file but keep metadata.
	codePath := CacheKeyPath(cacheDir, key) + ".code"
	if err := os.Remove(codePath); err != nil {
		t.Fatal(err)
	}

	// TryReadCache should return an error (missing code for committed entry).
	_, err = TryReadCache(cacheDir, key)
	if err == nil {
		t.Fatal("expected error for missing code")
	}

	// Metadata should also be cleaned up.
	metaPath := CacheKeyPath(cacheDir, key) + ".meta"
	if _, statErr := os.Stat(metaPath); !os.IsNotExist(statErr) {
		t.Fatal("meta was not cleaned up after missing code")
	}
}

func TestTryReadCacheStaleSchema(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()
	key := CacheKey(req, engine.Identity())

	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteCache(cacheDir, key, &result); err != nil {
		t.Fatal(err)
	}

	// Tamper with the cached metadata to set a wrong schema version.
	metaPath := CacheKeyPath(cacheDir, key) + ".meta"
	entry := cacheEntry{
		SchemaVersion: 999,
		CodeDigest:    digestBytes(result.Code),
		OutputDigest:  computeOutputDigest(result.Code, result.SourceMap),
	}
	data, _ := json.Marshal(entry)
	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// TryReadCache should return nil, nil (stale schema = clean miss).
	cached, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatalf("TryReadCache stale schema: %v", err)
	}
	if cached != nil {
		t.Fatal("expected nil for stale schema")
	}

	// Stale files should be removed.
	if _, statErr := os.Stat(metaPath); !os.IsNotExist(statErr) {
		t.Fatal("stale meta not removed")
	}
}

func TestTryReadCacheInvalidKeyShape(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	_, err := TryReadCache(cacheDir, "../etc/passwd")
	if err == nil {
		t.Fatal("expected error for invalid key shape")
	}
}

func TestWriteCacheNilResult(t *testing.T) {
	dir := t.TempDir()
	err := WriteCache(dir, "aaaa"+strings.Repeat("00", 30), nil)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestCacheKeyDeterminism(t *testing.T) {
	engine := testEngine()
	req := testRequest()

	k1 := CacheKey(req, engine.Identity())
	k2 := CacheKey(req, engine.Identity())
	if k1 != k2 {
		t.Fatal("non-deterministic cache key")
	}

	// Different source → different key.
	req2 := testRequest()
	req2.SourceBytes = []byte("different source")
	k3 := CacheKey(req2, engine.Identity())
	if k1 == k3 {
		t.Fatal("different sources produced same cache key")
	}
}

func TestCacheKeyIncludesSourceMapMode(t *testing.T) {
	engine := testEngine()
	req1 := testRequest()
	req1.SourceMapMode = SourceMapNone
	req2 := testRequest()
	req2.SourceMapMode = SourceMapInline

	k1 := CacheKey(req1, engine.Identity())
	k2 := CacheKey(req2, engine.Identity())
	if k1 == k2 {
		t.Fatal("source map mode not reflected in cache key")
	}
}

func TestComputeOutputDigest(t *testing.T) {
	code := []byte("const x = 1;\n")
	srcMap := []byte(`{"version":3}`)

	d1 := computeOutputDigest(code, srcMap)
	d2 := computeOutputDigest(code, srcMap)
	if d1 != d2 {
		t.Fatal("non-deterministic output digest")
	}

	// Different code → different digest.
	d3 := computeOutputDigest([]byte("const y = 2;\n"), srcMap)
	if d1 == d3 {
		t.Fatal("different code produced same digest")
	}

	// Different map → different digest.
	d4 := computeOutputDigest(code, []byte(`{"version":3,"sources":["a.ts"]}`))
	if d1 == d4 {
		t.Fatal("different map produced same digest")
	}
}

func TestCacheEntryRoundTripWithMap(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()
	req.SourceMapMode = SourceMapExternal

	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.SourceMap) == 0 {
		t.Fatal("no source map generated")
	}

	key := CacheKey(req, engine.Identity())
	if err := WriteCache(cacheDir, key, &result); err != nil {
		t.Fatal(err)
	}

	cached, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatalf("TryReadCache with map: %v", err)
	}
	if cached == nil {
		t.Fatal("expected cache hit")
	}
	if !VerifyCachedResult(cached, &result) {
		t.Fatal("source map not preserved in cache")
	}
}

// --- Cache failure classification ---

func TestWriteCacheDirCreationFailure(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file where the cache prefix directory should be.
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)
	key := "aaaa" + strings.Repeat("00", 30)
	prefixDir := filepath.Dir(CacheKeyPath(cacheDir, key))
	// Place a file at the prefix directory path so MkdirAll fails.
	if err := os.MkdirAll(filepath.Dir(prefixDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefixDir, []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := testEngine()
	req := testRequest()
	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	err = WriteCache(cacheDir, key, &result)
	if err == nil {
		t.Fatal("expected error when cache dir creation fails")
	}
	if apperr.CodeOf(err) != apperr.IO {
		t.Fatalf("expected IO code, got %s", apperr.CodeOf(err))
	}
}

func TestWriteCacheReadOnlyDir(t *testing.T) {
	// os.Chmod 0o555 does not prevent file creation inside the directory on
	// Windows; the Unix permission model the test relies on is unavailable.
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory semantics not available on Windows")
	}
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)
	key := "aaaa" + strings.Repeat("00", 30)
	entryDir := filepath.Dir(CacheKeyPath(cacheDir, key))
	// Create the entry dir with write permission first.
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Make it read-only so WriteCache fails on writeAtomic.
	if err := os.Chmod(entryDir, 0o555); err != nil {
		t.Fatal(err)
	}
	// Restore write permission at the end so TempDir cleanup works.
	defer func() { _ = os.Chmod(entryDir, 0o755) }()

	engine := testEngine()
	req := testRequest()
	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	err = WriteCache(cacheDir, key, &result)
	if err == nil {
		t.Fatal("expected error when cache dir is read-only")
	}
}

func TestClassifyCacheIOErrorPermission(t *testing.T) {
	err := classifyCacheIOError(os.ErrPermission, "write-code", "testkey")
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.IO {
		t.Fatalf("expected IO code, got %s", apperr.CodeOf(err))
	}
}

func TestClassifyCacheIOErrorPathPermission(t *testing.T) {
	pathErr := &os.PathError{Op: "write", Path: "/cache/test", Err: os.ErrPermission}
	err := classifyCacheIOError(pathErr, "write-code", "testkey")
	if apperr.CodeOf(err) != apperr.IO {
		t.Fatalf("expected IO code, got %s", apperr.CodeOf(err))
	}
}

func TestClassifyCacheIOErrorDiskFull(t *testing.T) {
	// Simulate a "no space left on device" path error.
	pathErr := &os.PathError{Op: "write", Path: "/cache/test", Err: os.NewSyscallError("write", syscallENOSPC())}
	err := classifyCacheIOError(pathErr, "write-code", "testkey")
	if apperr.CodeOf(err) != apperr.IO {
		t.Fatalf("expected IO code, got %s", apperr.CodeOf(err))
	}
}

// syscallENOSPC returns a fake no-space error without importing syscall directly.
func syscallENOSPC() error {
	return fakeErr("no space left on device")
}

type fakeErr string

func (e fakeErr) Error() string { return string(e) }
func (e fakeErr) Timeout() bool { return false }

func TestClassifyCacheIOErrorNil(t *testing.T) {
	err := classifyCacheIOError(nil, "write", "k")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// --- Recoverable corruption still regenerates ---

func TestWriteCacheFailureDoesNotAffectCorruptionRecovery(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	cacheDir := TransformCacheDir(eff)

	engine := testEngine()
	req := testRequest()
	key := CacheKey(req, engine.Identity())

	result, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}

	// First: write cache successfully.
	if err := WriteCache(cacheDir, key, &result); err != nil {
		t.Fatal(err)
	}

	// Verify it's readable.
	cached, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatalf("TryReadCache after write: %v", err)
	}
	if cached == nil {
		t.Fatal("expected cache hit")
	}
	if !VerifyCachedResult(cached, &result) {
		t.Fatal("cache result mismatch")
	}

	// Now corrupt the code (digest mismatch).
	codePath := CacheKeyPath(cacheDir, key) + ".code"
	if err := os.WriteFile(codePath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}

	// TryReadCache should detect corruption, clean up, and return an error.
	_, err = TryReadCache(cacheDir, key)
	if err == nil {
		t.Fatal("expected corruption error")
	}

	// After corruption cleanup, a fresh WriteCache + TryReadCache should work.
	result2, err := engine.Transform(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteCache(cacheDir, key, &result2); err != nil {
		t.Fatal(err)
	}

	cached2, err := TryReadCache(cacheDir, key)
	if err != nil {
		t.Fatalf("TryReadCache after corruption recovery: %v", err)
	}
	if cached2 == nil {
		t.Fatal("expected cache hit after recovery")
	}
	if !VerifyCachedResult(cached2, &result2) {
		t.Fatal("recovered cache result mismatch")
	}
}
