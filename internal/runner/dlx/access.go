package dlx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/fsx"
)

const accessSchemaVersion = 1

// AccessRecord is the best-effort access sidecar for an environment.
type AccessRecord struct {
	SchemaVersion     int    `json:"schemaVersion"`
	EnvironmentDigest string `json:"environmentDigest"`
	LastUsedAt        string `json:"lastUsedAt"`
	LastVerifiedAt    string `json:"lastVerifiedAt,omitempty"`
}

// AccessPath returns the access sidecar path for an environment digest.
func AccessPath(mxCacheDir, digest string) string {
	return filepath.Join(mxCacheDir, "access", digest+".json")
}

// TouchAccess updates lastUsedAt for an environment (best effort).
func TouchAccess(mxCacheDir, digest string) {
	path := AccessPath(mxCacheDir, digest)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rec := AccessRecord{
		SchemaVersion:     accessSchemaVersion,
		EnvironmentDigest: digest,
		LastUsedAt:        now,
		LastVerifiedAt:    now,
	}
	if b, err := os.ReadFile(path); err == nil {
		var prior AccessRecord
		if json.Unmarshal(b, &prior) == nil && prior.LastVerifiedAt != "" {
			rec.LastVerifiedAt = prior.LastVerifiedAt
		}
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return
	}
	data = append(data, '\n')
	_ = fsx.WriteAtomic(path, data, 0o644)
}
