package dlx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// QuarantineMeta records why an environment was quarantined.
type QuarantineMeta struct {
	SchemaVersion     int    `json:"schemaVersion"`
	EnvironmentDigest string `json:"environmentDigest"`
	Failure           string `json:"failure"`
	QuarantinedAt     string `json:"quarantinedAt"`
}

// QuarantineEnvironment moves a corrupt environment aside.
func QuarantineEnvironment(mxCacheDir, envDir, digest, reason string) (string, error) {
	if _, err := os.Stat(envDir); err != nil {
		return "", apperr.Wrap(apperr.NotFound, "dlx.quarantine", envDir, err)
	}
	id, err := fsx.NewLockID()
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "dlx.quarantine", digest, err)
	}
	dest := filepath.Join(mxCacheDir, "quarantine", fmt.Sprintf("%s.%d.%s", digest, time.Now().Unix(), id))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", apperr.Wrap(apperr.IO, "dlx.quarantine", dest, err)
	}
	meta := QuarantineMeta{
		SchemaVersion:     1,
		EnvironmentDigest: digest,
		Failure:           reason,
		QuarantinedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}
	b, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(envDir, "quarantine.json"), append(b, '\n'), 0o644); err != nil {
		return "", apperr.Wrap(apperr.IO, "dlx.quarantine", envDir, err)
	}
	if err := os.Rename(envDir, dest); err != nil {
		return "", apperr.Wrap(apperr.IO, "dlx.quarantine", dest, err)
	}
	return dest, nil
}
