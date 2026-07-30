package dlx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

const leaseSchemaVersion = 1

// ExecutionLease protects active environment use from prune.
type ExecutionLease struct {
	SchemaVersion     int    `json:"schemaVersion"`
	EnvironmentDigest string `json:"environmentDigest"`
	PID               int    `json:"pid"`
	ProcessStart      int64  `json:"processStart"`
	CreatedAt         string `json:"createdAt"`
	OwnerToken        string `json:"ownerToken"`
}

// LeaseDir returns the lease directory for an environment digest.
func LeaseDir(mxCacheDir, digest string) string {
	return filepath.Join(mxCacheDir, "leases", digest)
}

// AcquireExecutionLease creates a lease for the current process.
func AcquireExecutionLease(mxCacheDir, digest, ownerToken string, pid int, processStart int64) (string, func(), error) {
	dir := LeaseDir(mxCacheDir, digest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, apperr.Wrap(apperr.IO, "dlx.lease", dir, err)
	}
	id, err := fsx.NewLockID()
	if err != nil {
		return "", nil, apperr.Wrap(apperr.IO, "dlx.lease", digest, err)
	}
	path := filepath.Join(dir, id+".lease")
	lease := ExecutionLease{
		SchemaVersion:     leaseSchemaVersion,
		EnvironmentDigest: digest,
		PID:               pid,
		ProcessStart:      processStart,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339Nano),
		OwnerToken:        ownerToken,
	}
	b, err := json.MarshalIndent(lease, "", "  ")
	if err != nil {
		return "", nil, apperr.Wrap(apperr.Internal, "dlx.lease", path, err)
	}
	b = append(b, '\n')
	if err := fsx.WriteGenerationExclusive(path, b, 0o644); err != nil {
		return "", nil, apperr.Wrap(apperr.IO, "dlx.lease", path, err)
	}
	release := func() { _ = os.Remove(path) }
	return path, release, nil
}

// HasLiveLeases reports whether any lease files exist for digest.
func HasLiveLeases(mxCacheDir, digest string) bool {
	dir := LeaseDir(mxCacheDir, digest)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(ents) > 0
}
