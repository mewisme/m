package dlx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// StagingDir returns a unique staging directory for environment publication.
func StagingDir(mxCacheDir, envDigest, txnID string) string {
	return filepath.Join(mxCacheDir, "exec", envDigest+".staging."+txnID)
}

// PublishEnvironment atomically renames staging to the final environment directory.
func PublishEnvironment(stagingDir, finalDir string, ready ReadyMarker) error {
	if ready.SchemaVersion == 0 {
		ready.SchemaVersion = readySchemaVersion
	}
	if ready.CreatedAt == "" {
		ready.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	b, err := json.MarshalIndent(ready, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "dlx.publish", finalDir, err)
	}
	b = append(b, '\n')
	if err := fsx.WriteAtomic(ReadyPath(stagingDir), b, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "dlx.publish", finalDir, err)
	}
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "dlx.publish", finalDir, err)
	}
	if err := fsx.PublishRename(stagingDir, finalDir); err != nil {
		return apperr.Wrap(apperr.IO, "dlx.publish", finalDir, err)
	}
	return nil
}
