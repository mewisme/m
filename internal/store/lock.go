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

// importLockReleaseTestHook is set by tests to simulate post-success release failure.
var importLockReleaseTestHook func(lockDir string) error

// indexLockReleaseTestHook is set by tests to simulate post-success release failure.
var indexLockReleaseTestHook func(lockDir string) error

// SetImportLockReleaseTestHook installs a test hook for import lock release (test-only).
func SetImportLockReleaseTestHook(fn func(lockDir string) error) {
	importLockReleaseTestHook = fn
}

// SetIndexLockReleaseTestHook installs a test hook for index lock release (test-only).
func SetIndexLockReleaseTestHook(fn func(lockDir string) error) {
	indexLockReleaseTestHook = fn
}

func releaseStoreDirLock(op, lockDir string, match func([]byte) bool) error {
	result, err := fsx.ReleaseDirLock(lockDir, match)
	if err != nil {
		return apperr.Wrap(apperr.Store, op, lockDir, err)
	}
	switch result {
	case fsx.ReleaseOK:
		return nil
	case fsx.ReleaseNotOwner:
		return apperr.New(apperr.Store, op, lockDir, "lock not released: not owner")
	case fsx.ReleaseMissingOwner:
		return apperr.New(apperr.Store, op, lockDir, "lock not released: missing owner")
	case fsx.ReleaseMalformedOwner:
		return apperr.New(apperr.Store, op, lockDir, "lock not released: malformed owner")
	default:
		return nil
	}
}

func acquireImportLock(ctx context.Context, storeRoot string, key PackageKey) (release func() error, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	lockDir := externalImportLockPath(storeRoot, key)
	if err := removeLegacyImportLockFile(storeRoot, key); err != nil {
		return nil, apperr.Wrap(apperr.Store, "store.import.lock", key.String(), err)
	}

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
	_, err = fsx.AcquireDirLock(ctx, lockDir, raw, fsx.DirLockOptions{
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
	return func() error {
		if importLockReleaseTestHook != nil {
			if hookErr := importLockReleaseTestHook(lockDir); hookErr != nil {
				return hookErr
			}
		}
		return releaseStoreDirLock("store.import.lock.release", lockDir, match)
	}, nil
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

func removeLegacyImportLockFile(storeRoot string, key PackageKey) error {
	legacy := legacyImportLockFilePath(storeRoot, key)
	data, err := os.ReadFile(legacy)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	doc, err := parseImportLockDocument(data)
	if err != nil || !importLockHolderAlive(doc) {
		info, statErr := os.Stat(legacy)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil
			}
			return statErr
		}
		obs := fsx.ObservationFromOwner(data, info.ModTime(), false)
		tomb := fsx.TombstoneRoot(externalImportLockPath(storeRoot, key))
		if err := fsx.TakeoverStaleFileLock(legacy, obs, tomb); err != nil {
			if errors.Is(err, os.ErrExist) {
				return nil
			}
			return err
		}
	}
	return nil
}

// IndexLockDocument is persisted at <store>/.locks/index/owner.json.
type IndexLockDocument struct {
	SchemaVersion int       `json:"schemaVersion"`
	LockID        string    `json:"lockId"`
	PID           int       `json:"pid"`
	ProcessStart  int64     `json:"processStart"`
	CreatedAt     time.Time `json:"createdAt"`
}

func indexLockDir(storeRoot string) string {
	return filepath.Join(storeRoot, ".locks", "index")
}

func acquireIndexLock(ctx context.Context, storeRoot string) (release func() error, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	lockDir := indexLockDir(storeRoot)
	doc, raw, err := newIndexLockDocument()
	if err != nil {
		return nil, err
	}
	match := func(data []byte) bool { return indexLockOwnerMatches(data, doc.LockID) }
	stale := func(data []byte, mod time.Time) bool {
		if len(data) > 0 {
			parsed, err := parseIndexLockDocument(data)
			if err == nil {
				return !indexLockHolderAlive(parsed)
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
	_, err = fsx.AcquireDirLock(ctx, lockDir, raw, fsx.DirLockOptions{
		RetryInterval: importLockRetryInterval,
		MaxWait:       importLockMaxWait,
		GracePeriod:   importLockGracePeriod,
	}, stale, match)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, apperr.New(apperr.Store, "store.index.lock", lockDir,
				"timeout waiting for index lock")
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, apperr.Wrap(apperr.Cancelled, "store.index.lock", lockDir, err)
		}
		return nil, apperr.Wrap(apperr.Store, "store.index.lock", lockDir, err)
	}
	return func() error {
		if indexLockReleaseTestHook != nil {
			if hookErr := indexLockReleaseTestHook(lockDir); hookErr != nil {
				return hookErr
			}
		}
		return releaseStoreDirLock("store.index.lock.release", lockDir, match)
	}, nil
}

func newIndexLockDocument() (IndexLockDocument, []byte, error) {
	pid, start, err := currentProcessIdentity()
	if err != nil {
		return IndexLockDocument{}, nil, apperr.Wrap(apperr.Store, "store.index.lock", "", err)
	}
	lockID, err := fsx.NewLockID()
	if err != nil {
		return IndexLockDocument{}, nil, apperr.Wrap(apperr.Store, "store.index.lock", "", err)
	}
	doc := IndexLockDocument{
		SchemaVersion: importLockSchemaVersion,
		LockID:        lockID,
		PID:           pid,
		ProcessStart:  start,
		CreatedAt:     time.Now().UTC(),
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return IndexLockDocument{}, nil, apperr.Wrap(apperr.Store, "store.index.lock", "", err)
	}
	raw = append(raw, '\n')
	return doc, raw, nil
}

func parseIndexLockDocument(data []byte) (IndexLockDocument, error) {
	var doc IndexLockDocument
	if len(data) == 0 {
		return IndexLockDocument{}, apperr.New(apperr.Store, "store.index.lock", "", "empty lock owner")
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return IndexLockDocument{}, apperr.Wrap(apperr.Store, "store.index.lock", "", err)
	}
	return doc, nil
}

func indexLockHolderAlive(doc IndexLockDocument) bool {
	if doc.PID <= 0 {
		return false
	}
	if doc.SchemaVersion >= 2 && doc.ProcessStart != 0 {
		return processIdentityAlive(doc.PID, doc.ProcessStart)
	}
	return processAlive(doc.PID)
}

func indexLockOwnerMatches(data []byte, lockID string) bool {
	doc, err := parseIndexLockDocument(data)
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
