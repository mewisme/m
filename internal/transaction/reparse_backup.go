package transaction

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
)

const reparseSidecarSuffix = ".reparse.json"

type reparseBackupMeta struct {
	Tag        uint32 `json:"tag"`
	Substitute string `json:"substitute"`
	Print      string `json:"print"`
}

func writeReparseSidecar(path string, meta reparseBackupMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.backup", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.backup", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.backup", path, err)
	}
	return nil
}

func readReparseSidecar(path string) (reparseBackupMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return reparseBackupMeta{}, apperr.Wrap(apperr.IO, "transaction.restore", path, err)
	}
	var meta reparseBackupMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return reparseBackupMeta{}, apperr.Wrap(apperr.IO, "transaction.restore", path, err)
	}
	return meta, nil
}
