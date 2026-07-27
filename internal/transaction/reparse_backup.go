package transaction

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
)

const (
	backupsMetaDir           = "backups-meta"
	reparseMetaSchemaVersion = 1
	reparseEntryTypeJunction = "junction"
)

type reparseBackupMeta struct {
	SchemaVersion int    `json:"schemaVersion"`
	RelPath       string `json:"relPath"`
	Tag           uint32 `json:"tag"`
	Substitute    string `json:"substitute"`
	Print         string `json:"print"`
	EntryType     string `json:"entryType"`
}

func reparseMetaBackupRel(relPath string) string {
	return filepath.ToSlash(filepath.Join(backupsMetaDir, relPath+".json"))
}

func reparseMetaFilePath(metaRoot, relPath string) string {
	return filepath.Join(metaRoot, filepath.FromSlash(relPath)+".json")
}

func validateReparseRelPath(relPath string) error {
	if relPath == "" {
		return apperr.New(apperr.Transaction, "transaction.reparse", "", "empty relPath")
	}
	if filepath.IsAbs(relPath) {
		return apperr.New(apperr.Transaction, "transaction.reparse", relPath, "absolute relPath not allowed")
	}
	slash := filepath.ToSlash(relPath)
	if slash != relPath {
		return apperr.New(apperr.Transaction, "transaction.reparse", relPath, "relPath must use forward slashes")
	}
	if strings.Contains(relPath, "..") {
		return apperr.New(apperr.Transaction, "transaction.reparse", relPath, "relPath must not contain ..")
	}
	clean := filepath.Clean(filepath.FromSlash(relPath))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return apperr.New(apperr.Transaction, "transaction.reparse", relPath, "relPath escapes project root")
	}
	return nil
}

func validateReparseBackupMeta(meta reparseBackupMeta, projectRoot string) error {
	if meta.SchemaVersion != reparseMetaSchemaVersion {
		return apperr.New(apperr.Transaction, "transaction.reparse", "",
			fmt.Sprintf("unsupported schema version %d", meta.SchemaVersion))
	}
	if meta.Tag != fsx.IOReparseTagMountPoint {
		return apperr.New(apperr.Transaction, "transaction.reparse", meta.RelPath,
			fmt.Sprintf("unsupported reparse tag 0x%08X", meta.Tag))
	}
	if meta.EntryType != reparseEntryTypeJunction {
		return apperr.New(apperr.Transaction, "transaction.reparse", meta.RelPath,
			fmt.Sprintf("unsupported entry type %q", meta.EntryType))
	}
	if meta.Substitute == "" {
		return apperr.New(apperr.Transaction, "transaction.reparse", meta.RelPath, "empty substitute name")
	}
	if err := validateReparseRelPath(meta.RelPath); err != nil {
		return err
	}
	if _, err := GuardPath(projectRoot, meta.RelPath); err != nil {
		return err
	}
	return nil
}

func writeReparseMeta(metaRoot, relPath, substitute, print string) error {
	relPath = filepath.ToSlash(relPath)
	if err := validateReparseRelPath(relPath); err != nil {
		return err
	}
	meta := reparseBackupMeta{
		SchemaVersion: reparseMetaSchemaVersion,
		RelPath:       relPath,
		Tag:           fsx.IOReparseTagMountPoint,
		Substitute:    substitute,
		Print:         print,
		EntryType:     reparseEntryTypeJunction,
	}
	path := reparseMetaFilePath(metaRoot, relPath)
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

func readReparseMeta(path, projectRoot string) (reparseBackupMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return reparseBackupMeta{}, apperr.Wrap(apperr.IO, "transaction.restore", path, err)
	}
	var meta reparseBackupMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return reparseBackupMeta{}, apperr.Wrap(apperr.IO, "transaction.restore", path, err)
	}
	if err := validateReparseBackupMeta(meta, projectRoot); err != nil {
		return reparseBackupMeta{}, err
	}
	return meta, nil
}

func restoreJunctionMetas(metaRoot, projectRoot, relPrefix string) error {
	if metaRoot == "" {
		return nil
	}
	if _, err := os.Stat(metaRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "transaction.restore", metaRoot, err)
	}
	prefix := filepath.ToSlash(relPrefix)
	return filepath.WalkDir(metaRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		meta, err := readReparseMeta(path, projectRoot)
		if err != nil {
			return err
		}
		if prefix != "" && meta.RelPath != prefix && !strings.HasPrefix(meta.RelPath, prefix+"/") {
			return nil
		}
		live, err := GuardPath(projectRoot, meta.RelPath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.restore", live, err)
		}
		if err := os.RemoveAll(live); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.restore", live, err)
		}
		return createJunction(live, meta.Substitute, meta.Print)
	})
}
