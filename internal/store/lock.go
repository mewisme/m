package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

const (
	importLockSchemaVersion = 1
	importLockWaitTimeout   = 30 * time.Second
	importLockPollInterval  = 50 * time.Millisecond
)

// ImportLockDocument is persisted at <store>/.locks/<algo>/<hex>.lock.
type ImportLockDocument struct {
	SchemaVersion int       `json:"schemaVersion"`
	PID           int       `json:"pid"`
	StartedAt     time.Time `json:"startedAt"`
}

func externalImportLockPath(storeRoot string, key PackageKey) string {
	return filepath.Join(storeRoot, ".locks", key.Algo, key.Hex+".lock")
}

// HasImportLock reports whether an external import lock exists for key.
func HasImportLock(s *PackageStore, key PackageKey) bool {
	if s == nil || s.Root == "" {
		return false
	}
	_, err := os.Stat(externalImportLockPath(s.Root, key))
	return err == nil
}

func acquireImportLock(ctx context.Context, storeRoot string, key PackageKey) (release func(), err error) {
	lockPath := externalImportLockPath(storeRoot, key)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, apperr.Wrap(apperr.Store, "store.import.lock", lockPath, err)
	}
	deadline := time.Now().Add(importLockWaitTimeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, apperr.Wrap(apperr.Cancelled, "store.import.lock", key.String(), err)
		}
		release, acquired, err := tryAcquireImportLock(lockPath, key)
		if err != nil {
			return nil, err
		}
		if acquired {
			return release, nil
		}
		if time.Now().After(deadline) {
			return nil, apperr.New(apperr.Store, "store.import.lock", key.String(),
				"timeout waiting for import lock")
		}
		select {
		case <-ctx.Done():
			return nil, apperr.Wrap(apperr.Cancelled, "store.import.lock", key.String(), ctx.Err())
		case <-time.After(importLockPollInterval):
		}
	}
}

func tryAcquireImportLock(lockPath string, key PackageKey) (release func(), acquired bool, err error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		doc := ImportLockDocument{
			SchemaVersion: importLockSchemaVersion,
			PID:           os.Getpid(),
			StartedAt:     time.Now().UTC(),
		}
		raw, marshalErr := json.MarshalIndent(doc, "", "  ")
		if marshalErr != nil {
			_ = f.Close()
			_ = os.Remove(lockPath)
			return nil, false, apperr.Wrap(apperr.Store, "store.import.lock", key.String(), marshalErr)
		}
		raw = append(raw, '\n')
		if _, writeErr := f.Write(raw); writeErr != nil {
			_ = f.Close()
			_ = os.Remove(lockPath)
			return nil, false, apperr.Wrap(apperr.Store, "store.import.lock", key.String(), writeErr)
		}
		_ = f.Close()
		return func() { releaseImportLock(lockPath) }, true, nil
	}
	if !os.IsExist(err) {
		return nil, false, apperr.Wrap(apperr.Store, "store.import.lock", lockPath, err)
	}
	if recoverStaleImportLock(lockPath) {
		return tryAcquireImportLock(lockPath, key)
	}
	return nil, false, nil
}

func recoverStaleImportLock(lockPath string) bool {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return os.IsNotExist(err)
	}
	var doc ImportLockDocument
	if err := json.Unmarshal(data, &doc); err != nil || doc.PID <= 0 {
		_ = os.Remove(lockPath)
		return true
	}
	if importLockProcessAlive(doc.PID) {
		return false
	}
	_ = os.Remove(lockPath)
	return true
}

func releaseImportLock(lockPath string) {
	data, err := os.ReadFile(lockPath)
	if err == nil {
		var doc ImportLockDocument
		if err := json.Unmarshal(data, &doc); err == nil && doc.PID != 0 && doc.PID != os.Getpid() {
			return
		}
	}
	_ = os.Remove(lockPath)
}
