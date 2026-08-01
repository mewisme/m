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
	"sync"

	"github.com/mewisme/mew/internal/config"
)

// CacheSchemaVersion is bumped when the cache entry format changes.
const CacheSchemaVersion = 1

// cacheEntry is the serialized format stored on disk.
type cacheEntry struct {
	SchemaVersion int    `json:"schema_version"`
	CodeDigest    string `json:"code_digest"`
	MapDigest     string `json:"map_digest,omitempty"`
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

var cacheMu sync.Mutex

// TryReadCache attempts to read a cached transform result.
// Returns nil if the entry doesn't exist, is corrupt, or doesn't match.
func TryReadCache(cacheDir, key string) (*TransformResult, error) {
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
			return nil, nil // miss
		}
		return nil, fmt.Errorf("reading cache meta: %w", metaErr)
	}

	var entry cacheEntry
	if err := json.Unmarshal(metaData, &entry); err != nil {
		_ = os.Remove(metaPath)
		return nil, fmt.Errorf("corrupt cache meta: %w", err)
	}
	if entry.SchemaVersion != CacheSchemaVersion {
		return nil, nil // stale schema → miss
	}

	// Read code.
	code, codeErr := os.ReadFile(codePath)
	if codeErr != nil {
		if os.IsNotExist(codeErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache code: %w", codeErr)
	}
	if digestBytes(code) != entry.CodeDigest {
		_ = os.Remove(codePath)
		_ = os.Remove(mapPath)
		_ = os.Remove(metaPath)
		return nil, fmt.Errorf("cache code digest mismatch at %s", key)
	}

	// Read source map if present.
	var srcMap []byte
	if entry.MapDigest != "" {
		srcMap, codeErr = os.ReadFile(mapPath)
		if codeErr != nil && !os.IsNotExist(codeErr) {
			return nil, fmt.Errorf("reading cache map: %w", codeErr)
		}
		if len(srcMap) > 0 && digestBytes(srcMap) != entry.MapDigest {
			_ = os.Remove(mapPath)
			_ = os.Remove(metaPath)
			return nil, fmt.Errorf("cache map digest mismatch at %s", key)
		}
	}

	return &TransformResult{
		Code:         code,
		SourceMap:    srcMap,
		OutputDigest: entry.CodeDigest,
		CacheStatus:  CacheStatusHit,
		Elapsed:      0,
	}, nil
}

// WriteCache writes a transform result to cache atomically.
func WriteCache(cacheDir, key string, result *TransformResult) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	entryPath := CacheKeyPath(cacheDir, key)
	codePath := entryPath + ".code"
	mapPath := entryPath + ".map"
	metaPath := entryPath + ".meta"

	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		return fmt.Errorf("mkdir cache: %w", err)
	}

	entry := cacheEntry{
		SchemaVersion: CacheSchemaVersion,
		CodeDigest:    digestBytes(result.Code),
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
