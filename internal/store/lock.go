package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
)

const (
	importLockSchemaVersion = 2
	importLockRetryInterval = 25 * time.Millisecond
	importLockMaxWait       = 30 * time.Second
	importLockGracePeriod   = fsx.DefaultLockGrace
)

// ImportLockDocument is persisted at <store>/.locks/<algo>/<hex>/owner.json (schema v2).
type ImportLockDocument struct {
	SchemaVersion int       `json:"schemaVersion"`
	LockID        string    `json:"lockId"`
	PID           int       `json:"pid"`
	ProcessStart  int64     `json:"processStart"`
	PackageKey    string    `json:"packageKey"`
	CreatedAt     time.Time `json:"createdAt"`
	// StartedAt is the v1 field; kept for stale-lock reads only.
	StartedAt time.Time `json:"startedAt,omitempty"`
}

func externalImportLockPath(storeRoot string, key PackageKey) string {
	return filepath.Join(storeRoot, ".locks", key.Algo, key.Hex)
}

func legacyImportLockFilePath(storeRoot string, key PackageKey) string {
	return filepath.Join(storeRoot, ".locks", key.Algo, key.Hex+".lock")
}

// HasImportLock reports whether an external import lock exists for key.
func HasImportLock(s *PackageStore, key PackageKey) bool {
	if s == nil || s.Root == "" {
		return false
	}
	if _, err := os.Stat(externalImportLockPath(s.Root, key)); err == nil {
		return true
	}
	_, err := os.Stat(legacyImportLockFilePath(s.Root, key))
	return err == nil
}

func acquireImportLock(ctx context.Context, storeRoot string, key PackageKey) (release func(), err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	lockDir := externalImportLockPath(storeRoot, key)
	removeLegacyImportLockFile(storeRoot, key)

	doc, raw, err := newImportLockDocument(key)
	if err != nil {
		return nil, err
	}
	match := func(data []byte) bool { return importLockOwnerMatches(data, doc.LockID) }
	stale := func(data []byte, mod time.Time) bool {
		if len(data) > 0 {
			parsed, err := parseImportLockDocument(data)
			if err == nil {
				return !importLockHolderAlive(parsed)
			}
			if time.Since(mod) < importLockGracePeriod {
				return false
			}
			return true
		}
		if time.Since(mod) < importLockGracePeriod {
			return false
		}
		return true
	}
	release, err = fsx.AcquireDirLock(ctx, lockDir, raw, fsx.DirLockOptions{
		RetryInterval: importLockRetryInterval,
		MaxWait:       importLockMaxWait,
		GracePeriod:   importLockGracePeriod,
	}, stale, match)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, apperr.New(apperr.Store, "store.import.lock", key.String(),
				"timeout waiting for import lock")
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, apperr.Wrap(apperr.Cancelled, "store.import.lock", key.String(), err)
		}
		return nil, apperr.Wrap(apperr.Store, "store.import.lock", lockDir, err)
	}
	return release, nil
}

func newImportLockDocument(key PackageKey) (ImportLockDocument, []byte, error) {
	pid, start, err := currentProcessIdentity()
	if err != nil {
		return ImportLockDocument{}, nil, apperr.Wrap(apperr.Store, "store.import.lock", key.String(), err)
	}
	lockID, err := fsx.NewLockID()
	if err != nil {
		return ImportLockDocument{}, nil, apperr.Wrap(apperr.Store, "store.import.lock", key.String(), err)
	}
	doc := ImportLockDocument{
		SchemaVersion: importLockSchemaVersion,
		LockID:        lockID,
		PID:           pid,
		ProcessStart:  start,
		PackageKey:    key.String(),
		CreatedAt:     time.Now().UTC(),
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return ImportLockDocument{}, nil, apperr.Wrap(apperr.Store, "store.import.lock", key.String(), err)
	}
	raw = append(raw, '\n')
	return doc, raw, nil
}

func parseImportLockDocument(data []byte) (ImportLockDocument, error) {
	var doc ImportLockDocument
	if len(data) == 0 {
		return ImportLockDocument{}, apperr.New(apperr.Store, "store.import.lock", "", "empty lock owner")
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ImportLockDocument{}, apperr.Wrap(apperr.Store, "store.import.lock", "", err)
	}
	if doc.SchemaVersion == 0 && !doc.StartedAt.IsZero() {
		doc.SchemaVersion = 1
	}
	return doc, nil
}

func importLockHolderAlive(doc ImportLockDocument) bool {
	if doc.PID <= 0 {
		return false
	}
	if doc.SchemaVersion >= 2 && doc.ProcessStart != 0 {
		return processIdentityAlive(doc.PID, doc.ProcessStart)
	}
	return processAlive(doc.PID)
}

func importLockOwnerMatches(data []byte, lockID string) bool {
	doc, err := parseImportLockDocument(data)
	if err != nil {
		return false
	}
	if lockID != "" && doc.LockID != lockID {
		return false
	}
	pid, start, err := currentProcessIdentity()
	if err != nil {
		return false
	}
	if doc.PID != pid {
		return false
	}
	if doc.SchemaVersion >= 2 && doc.ProcessStart != 0 {
		return doc.ProcessStart == start
	}
	return true
}

func removeLegacyImportLockFile(storeRoot string, key PackageKey) {
	legacy := legacyImportLockFilePath(storeRoot, key)
	data, err := os.ReadFile(legacy)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}
	doc, err := parseImportLockDocument(data)
	if err != nil || !importLockHolderAlive(doc) {
		_ = os.Remove(legacy)
	}
}
