package transform

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
)

// CacheSchemaVersion is bumped when the cache entry format changes.
const CacheSchemaVersion = 1

// cacheEntry is the serialized format stored on disk.
type cacheEntry struct {
	SchemaVersion int    `json:"schema_version"`
	CodeDigest    string `json:"code_digest"`
	MapDigest     string `json:"map_digest,omitempty"`
	OutputDigest  string `json:"output_digest"`
}

// TransformCacheDir returns the root of the transform cache.
func TransformCacheDir(eff *config.Effective) string {
	root := config.CacheRoot(eff)
	return filepath.Join(root, "transform", fmt.Sprintf("v%d", CacheSchemaVersion))
}

// CacheKey builds a stable SHA-256 key from transform inputs.
func CacheKey(req TransformRequest, identity EngineIdentity) string {
	h := sha256.New()
	h.Write(req.SourceBytes)
	h.Write([]byte(req.SourcePath))
	h.Write([]byte(req.Loader))
	h.Write([]byte(req.Format))
	optsData, _ := json.Marshal(req.NormalizedOpts)
	h.Write(optsData)
	h.Write([]byte(req.TsconfigDigest))
	h.Write([]byte(identity.Name))
	h.Write([]byte(identity.Version))
	h.Write([]byte(req.SourceMapMode))
	fmt.Fprintf(h, "%d", req.TargetNodeMajor)
	return hex.EncodeToString(h.Sum(nil))
}

// CacheKeyPath returns the filesystem path for a cache key.
func CacheKeyPath(cacheDir, key string) string {
	// Split key into prefix directories for filesystem friendliness.
	return filepath.Join(cacheDir, key[:2], key)
}

// keyShapeRE validates a cache key is a 64-char hex string (SHA-256).
var keyShapeRE = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

var cacheMu sync.Mutex

// TryReadCache attempts to read a cached transform result.
// Returns nil, nil for a clean miss (no entry exists).
// Returns an error for corruption, permission failure, or I/O errors.
func TryReadCache(cacheDir, key string) (*TransformResult, error) {
	// Validate key shape before any filesystem access.
	if !keyShapeRE.MatchString(key) {
		return nil, fmt.Errorf("invalid cache key shape %q", key)
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()

	entryPath := CacheKeyPath(cacheDir, key)
	codePath := entryPath + ".code"
	mapPath := entryPath + ".map"
	metaPath := entryPath + ".meta"

	// Read metadata.
	metaData, metaErr := os.ReadFile(metaPath)
	if metaErr != nil {
		if os.IsNotExist(metaErr) {
			return nil, nil // clean miss
		}
		// Permission or unexpected I/O: propagate, don't silently convert to miss.
		return nil, fmt.Errorf("reading cache meta: %w", metaErr)
	}

	var entry cacheEntry
	if err := json.Unmarshal(metaData, &entry); err != nil {
		_ = os.Remove(metaPath)
		return nil, fmt.Errorf("corrupt cache meta: %w", err)
	}
	if entry.SchemaVersion != CacheSchemaVersion {
		_ = os.Remove(metaPath)
		_ = os.Remove(codePath)
		_ = os.Remove(mapPath)
		return nil, nil // stale schema: clean up and miss
	}

	// Read code. Missing code when metadata references it is corruption.
	code, codeErr := os.ReadFile(codePath)
	if codeErr != nil {
		if os.IsNotExist(codeErr) {
			// Metadata committed but code missing: corrupt entry, clean up.
			_ = os.Remove(metaPath)
			_ = os.Remove(mapPath)
			return nil, fmt.Errorf("cache code missing for committed entry at %s", key)
		}
		// Permission or I/O error: propagate.
		return nil, fmt.Errorf("reading cache code: %w", codeErr)
	}
	if digestBytes(code) != entry.CodeDigest {
		_ = os.Remove(codePath)
		_ = os.Remove(mapPath)
		_ = os.Remove(metaPath)
		return nil, fmt.Errorf("cache code digest mismatch at %s", key)
	}

	// Read source map if metadata references one.
	var srcMap []byte
	if entry.MapDigest != "" {
		srcMap, codeErr = os.ReadFile(mapPath)
		if codeErr != nil {
			if os.IsNotExist(codeErr) {
				// Metadata committed but map missing: corrupt entry, clean up.
				_ = os.Remove(codePath)
				_ = os.Remove(metaPath)
				return nil, fmt.Errorf("cache map missing for committed entry at %s", key)
			}
			return nil, fmt.Errorf("reading cache map: %w", codeErr)
		}
		if digestBytes(srcMap) != entry.MapDigest {
			_ = os.Remove(codePath)
			_ = os.Remove(mapPath)
			_ = os.Remove(metaPath)
			return nil, fmt.Errorf("cache map digest mismatch at %s", key)
		}
	}

	// Validate combined output digest (same computation as engine.Transform).
	expectedOutput := computeOutputDigest(code, srcMap)
	if entry.OutputDigest != expectedOutput {
		_ = os.Remove(codePath)
		_ = os.Remove(mapPath)
		_ = os.Remove(metaPath)
		return nil, fmt.Errorf("cache output digest mismatch at %s", key)
	}

	return &TransformResult{
		Code:         code,
		SourceMap:    srcMap,
		OutputDigest: entry.OutputDigest,
		CacheStatus:  CacheStatusHit,
		Elapsed:      0,
	}, nil
}

// WriteCache writes a transform result to cache atomically.
// Code and map are written first; metadata (the commit record) is written last.
// Permission and I/O errors are propagated.
func WriteCache(cacheDir, key string, result *TransformResult) error {
	if result == nil {
		return apperr.New(apperr.TransformCacheCorrupt, "transform.cache", key, "nil result")
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()

	entryPath := CacheKeyPath(cacheDir, key)
	codePath := entryPath + ".code"
	mapPath := entryPath + ".map"
	metaPath := entryPath + ".meta"

	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		return fmt.Errorf("mkdir cache: %w", err)
	}

	// Canonical output digest: SHA-256(code || map), same as engine.Transform.
	outputDigest := computeOutputDigest(result.Code, result.SourceMap)

	entry := cacheEntry{
		SchemaVersion: CacheSchemaVersion,
		CodeDigest:    digestBytes(result.Code),
		OutputDigest:  outputDigest,
	}

	// Write code atomically.
	if err := writeAtomic(codePath, result.Code); err != nil {
		return err
	}

	// Write source map if present.
	if len(result.SourceMap) > 0 {
		entry.MapDigest = digestBytes(result.SourceMap)
		if err := writeAtomic(mapPath, result.SourceMap); err != nil {
			return err
		}
	}

	// Write metadata last (acts as a commit record).
	metaData, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return writeAtomic(metaPath, metaData)
}

// computeOutputDigest returns SHA-256(code || map), matching engine.Transform.
func computeOutputDigest(code, srcMap []byte) string {
	h := sha256.New()
	h.Write(code)
	h.Write(srcMap)
	return hex.EncodeToString(h.Sum(nil))
}

// writeAtomic writes data to a file using temp + rename for atomicity.
func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp-" + randomHexSuffix()
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// digestBytes returns the hex SHA-256 of data.
func digestBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func randomHexSuffix() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// VerifyCachedResult checks that the cached result matches the request.
// Returns true if the cached code is valid for the given request inputs.
func VerifyCachedResult(cached, fresh *TransformResult) bool {
	if cached == nil || fresh == nil {
		return false
	}
	return bytes.Equal(cached.Code, fresh.Code) &&
		bytes.Equal(cached.SourceMap, fresh.SourceMap)
}
