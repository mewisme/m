package dlx

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
)

const readySchemaVersion = 1

// ReadyMarker is the publication marker for a warm environment.
type ReadyMarker struct {
	SchemaVersion              int    `json:"schemaVersion"`
	EnvironmentDigest          string `json:"environmentDigest"`
	GraphDigest                string `json:"graphDigest"`
	TargetPlatformFingerprint  string `json:"targetPlatformFingerprint"`
	NodeFingerprint            string `json:"nodeFingerprint"`
	LifecyclePolicyFingerprint string `json:"lifecyclePolicyFingerprint"`
	CreatedAt                  string `json:"createdAt"`
}

// EnvironmentDir returns the final cache directory for an environment digest.
func EnvironmentDir(mxCacheDir, digest string) string {
	return filepath.Join(mxCacheDir, "exec", digest)
}

// ReadyPath returns ready.json inside an environment directory.
func ReadyPath(envDir string) string {
	return filepath.Join(envDir, "ready.json")
}

// MLockPath returns m.lock inside an environment directory.
func MLockPath(envDir string) string {
	return filepath.Join(envDir, "m.lock")
}

// BinsPath returns bins.v1.json inside node_modules.
func BinsPath(envDir string) string {
	return filepath.Join(envDir, "node_modules", ".mew", "bins.v1.json")
}

// VerifyWarmEnvironment checks that envDir is a complete warm cache entry.
func VerifyWarmEnvironment(envDir string, want ResolvedEnvironmentIdentity) error {
	readyPath := ReadyPath(envDir)
	b, err := os.ReadFile(readyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return apperr.Wrap(apperr.NotFound, "dlx.cache", envDir, err)
		}
		return apperr.Wrap(apperr.IO, "dlx.cache", readyPath, err)
	}
	var ready ReadyMarker
	if err := json.Unmarshal(b, &ready); err != nil {
		return apperr.Wrap(apperr.Integrity, "dlx.cache", readyPath, err)
	}
	if ready.SchemaVersion != readySchemaVersion {
		return apperr.New(apperr.Integrity, "dlx.cache", readyPath, "unsupported ready schema")
	}
	if ready.EnvironmentDigest != want.Digest() {
		return apperr.New(apperr.Integrity, "dlx.cache", envDir, "environment digest mismatch")
	}
	if ready.GraphDigest != want.GraphDigest {
		return apperr.New(apperr.Integrity, "dlx.cache", envDir, "graph digest mismatch")
	}
	if _, err := os.Stat(MLockPath(envDir)); err != nil {
		return apperr.Wrap(apperr.Integrity, "dlx.cache", MLockPath(envDir), err)
	}
	nm := filepath.Join(envDir, "node_modules")
	doc, err := binmeta.Read(nm)
	if err != nil {
		return apperr.Wrap(apperr.Integrity, "dlx.cache", BinsPath(envDir), err)
	}
	if doc == nil || len(doc.Records) == 0 {
		return apperr.New(apperr.Integrity, "dlx.cache", BinsPath(envDir), "missing bin metadata")
	}
	return nil
}

// IsWarm reports whether envDir exists and has ready.json.
func IsWarm(envDir string) bool {
	st, err := os.Stat(ReadyPath(envDir))
	return err == nil && !st.IsDir()
}
