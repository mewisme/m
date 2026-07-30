package envexec

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

const cacheLayoutVersion = "v1"

// SharedCacheDir returns <cacheRoot>/envexec/v1/<source>/<identityDigest>/.
func SharedCacheDir(cacheRoot string, source SourceKind, identityDigest string) string {
	return filepath.Join(cacheRoot, "envexec", cacheLayoutVersion, string(source), identityDigest)
}

// ReadyMarker is the publication marker for a shared cache entry.
type ReadyMarker struct {
	SchemaVersion  int    `json:"schemaVersion"`
	IdentityDigest string `json:"identityDigest"`
	GraphDigest    string `json:"graphDigest"`
	Source         string `json:"source"`
	TargetPlatform string `json:"targetPlatform"`
	CreatedAt      string `json:"createdAt"`
}

const readySchemaVersion = 1

// ReadyPath returns ready.json inside an environment directory.
func ReadyPath(envDir string) string {
	return filepath.Join(envDir, "ready.json")
}

// IsWarm reports whether envDir has a ready marker.
func IsWarm(envDir string) bool {
	st, err := os.Stat(ReadyPath(envDir))
	return err == nil && !st.IsDir()
}

// VerifyWarmEnvironment checks canonical artifacts agree with identity.
func VerifyWarmEnvironment(envDir string, id EnvironmentIdentity) error {
	readyPath := ReadyPath(envDir)
	b, err := os.ReadFile(readyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return apperr.Wrap(apperr.NotFound, "envexec.cache", envDir, err)
		}
		return apperr.Wrap(apperr.IO, "envexec.cache", readyPath, err)
	}
	var ready ReadyMarker
	if err := json.Unmarshal(b, &ready); err != nil {
		return apperr.Wrap(apperr.Integrity, "envexec.cache", readyPath, err)
	}
	if ready.SchemaVersion != readySchemaVersion {
		return apperr.New(apperr.Integrity, "envexec.cache", readyPath, "unsupported ready schema")
	}
	if ready.IdentityDigest != id.IdentityDigest() {
		return apperr.New(apperr.Integrity, "envexec.cache", envDir, "identity digest mismatch")
	}
	if ready.GraphDigest != id.GraphDigest {
		return apperr.New(apperr.Integrity, "envexec.cache", envDir, "graph digest mismatch")
	}
	if _, err := os.Stat(filepath.Join(envDir, "m.lock")); err != nil {
		return apperr.Wrap(apperr.Integrity, "envexec.cache", envDir, err)
	}
	nm := filepath.Join(envDir, "node_modules")
	if _, err := os.Stat(filepath.Join(nm, ".mew", "bins.v1.json")); err != nil {
		return apperr.Wrap(apperr.Integrity, "envexec.cache", envDir, err)
	}
	if _, err := os.Stat(filepath.Join(envDir, ".mew", "generation.json")); err != nil {
		return apperr.Wrap(apperr.Integrity, "envexec.cache", envDir, err)
	}
	return nil
}

// PublishEnvironment atomically publishes staging to finalDir with ready marker.
func PublishEnvironment(stagingDir, finalDir string, ready ReadyMarker) error {
	if ready.SchemaVersion == 0 {
		ready.SchemaVersion = readySchemaVersion
	}
	b, err := json.MarshalIndent(ready, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "envexec.publish", finalDir, err)
	}
	b = append(b, '\n')
	if err := fsx.WriteAtomic(ReadyPath(stagingDir), b, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "envexec.publish", finalDir, err)
	}
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "envexec.publish", finalDir, err)
	}
	if err := fsx.PublishRename(stagingDir, finalDir); err != nil {
		return apperr.Wrap(apperr.IO, "envexec.publish", finalDir, err)
	}
	return nil
}

// StagingDir returns a unique staging directory for cache publication.
func StagingDir(parent, identityDigest, txnID string) string {
	return filepath.Join(parent, identityDigest+".staging."+txnID)
}
