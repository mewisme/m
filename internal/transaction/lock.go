package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

const (
	lockDirName        = "lock"
	lockOwnerFile      = fsx.OwnerFileName
	lockSchemaVersion  = 3
	lockRetryInterval  = 25 * time.Millisecond
	lockMaxWaitDefault = 30 * time.Second
	lockGracePeriod    = fsx.DefaultLockGrace
)

// LockDocument is persisted at .mew/txn/lock/owner.json (schema v3).
type LockDocument struct {
	SchemaVersion int       `json:"schemaVersion,omitempty"`
	LockID        string    `json:"lockId,omitempty"`
	PID           int       `json:"pid"`
	ProcessStart  int64     `json:"processStart,omitempty"`
	TxnID         string    `json:"txnId"`
	CreatedAt     time.Time `json:"createdAt"`
	ProjectRoot   string    `json:"projectRoot,omitempty"`
	Hostname      string    `json:"hostname,omitempty"`
	// StartedAt is the v1 field; kept for stale-lock reads only.
	StartedAt time.Time `json:"startedAt,omitempty"`
}

// LockPath returns <project>/.mew/txn/lock (directory).
func LockPath(projectRoot string) string {
	return filepath.Join(TxnRoot(projectRoot), lockDirName)
}

// AcquireProjectLock creates the project lock exclusively, waiting until ctx is done.
func AcquireProjectLock(ctx context.Context, projectRoot, txnID string) error {
	_, err := acquireProjectLock(ctx, projectRoot, txnID, false)
	return err
}

// TakeoverProjectLock replaces a stale project lock with a new recovery identity.
func TakeoverProjectLock(ctx context.Context, projectRoot, recoveryTxnID string) error {
	_, err := acquireProjectLock(ctx, projectRoot, recoveryTxnID, true)
	return err
}

func acquireProjectLock(ctx context.Context, projectRoot, txnID string, takeover bool) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "transaction.lock", projectRoot, err)
	}
	lockDir := LockPath(absRoot)
	doc, raw, err := newLockDocument(absRoot, txnID)
	if err != nil {
		return "", err
	}
	match := func(data []byte) bool { return lockOwnerMatches(data, doc.LockID, txnID) }
	stale := func(data []byte, mod time.Time) bool {
		if len(data) > 0 {
			parsed, err := parseLockDocument(data)
			if err == nil {
				return !lockHolderAlive(parsed)
			}
			if time.Since(mod) < lockGracePeriod && !takeover {
				return false
			}
			return true
		}
		if time.Since(mod) < lockGracePeriod && !takeover {
			return false
		}
		return true
	}
	release, err := fsx.AcquireDirLock(ctx, lockDir, raw, fsx.DirLockOptions{
		RetryInterval: lockRetryInterval,
		MaxWait:       lockMaxWaitDefault,
		GracePeriod:   lockGracePeriod,
	}, stale, match)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", apperr.New(apperr.Transaction, "transaction.lock", lockDir,
				"another install transaction is in progress")
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", apperr.Wrap(apperr.Cancelled, "transaction.lock", lockDir, err)
		}
		return "", apperr.Wrap(apperr.IO, "transaction.lock", lockDir, err)
	}
	_ = release // directory lock held until ReleaseProjectLock
	return doc.LockID, nil
}

// ReleaseProjectLock removes the project lock when owned by this process and txnID.
// An empty txnID releases only when the on-disk lock matches this process identity.
func ReleaseProjectLock(projectRoot, txnID string) error {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.lock", projectRoot, err)
	}
	lockDir := LockPath(absRoot)
	match := func(data []byte) bool {
		doc, err := parseLockDocument(data)
		if err != nil {
			return false
		}
		if txnID != "" && doc.TxnID != txnID {
			return false
		}
		return processOwnsLock(doc)
	}
	result, err := fsx.ReleaseDirLock(lockDir, match)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.lock", lockDir, err)
	}
	switch result {
	case fsx.ReleaseOK:
		return nil
	case fsx.ReleaseMissingOwner:
		return apperr.New(apperr.Transaction, "transaction.lock", lockDir, "lock owner metadata missing")
	case fsx.ReleaseMalformedOwner:
		return apperr.New(apperr.Transaction, "transaction.lock", lockDir, "lock owner metadata malformed")
	case fsx.ReleaseNotOwner:
		return apperr.New(apperr.Transaction, "transaction.lock", lockDir, "lock not owned by this process")
	default:
		return apperr.New(apperr.Transaction, "transaction.lock", lockDir, "lock release failed")
	}
}

func newLockDocument(projectRoot, txnID string) (LockDocument, []byte, error) {
	pid, start, err := currentProcessIdentity()
	if err != nil {
		return LockDocument{}, nil, apperr.Wrap(apperr.Transaction, "transaction.lock", projectRoot, err)
	}
	lockID, err := newLockID()
	if err != nil {
		return LockDocument{}, nil, apperr.Wrap(apperr.Transaction, "transaction.lock", projectRoot, err)
	}
	host, _ := os.Hostname()
	doc := LockDocument{
		SchemaVersion: lockSchemaVersion,
		LockID:        lockID,
		PID:           pid,
		ProcessStart:  start,
		TxnID:         txnID,
		CreatedAt:     time.Now().UTC(),
		ProjectRoot:   projectRoot,
		Hostname:      host,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return LockDocument{}, nil, apperr.Wrap(apperr.Transaction, "transaction.lock", projectRoot, err)
	}
	raw = append(raw, '\n')
	return doc, raw, nil
}

func newLockID() (string, error) {
	return fsx.NewLockID()
}

func parseLockDocument(data []byte) (LockDocument, error) {
	var doc LockDocument
	if len(data) == 0 {
		return LockDocument{}, apperr.New(apperr.Transaction, "transaction.lock", "", "empty lock owner")
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return LockDocument{}, apperr.Wrap(apperr.Transaction, "transaction.lock", "", err)
	}
	if doc.SchemaVersion == 0 && !doc.StartedAt.IsZero() {
		doc.SchemaVersion = 1
	}
	return doc, nil
}

func lockOwnerMatches(data []byte, lockID, txnID string) bool {
	doc, err := parseLockDocument(data)
	if err != nil {
		return false
	}
	if lockID != "" && doc.LockID != lockID {
		return false
	}
	if txnID != "" && doc.TxnID != txnID {
		return false
	}
	return processOwnsLock(doc)
}

func lockHolderAlive(doc LockDocument) bool {
	if doc.PID <= 0 {
		return false
	}
	if doc.SchemaVersion >= 2 && doc.ProcessStart != 0 {
		return processIdentityAlive(doc.PID, doc.ProcessStart)
	}
	return processAlive(doc.PID)
}

func processOwnsLock(doc LockDocument) bool {
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

// tryRemoveStaleLock is kept for LoadIncomplete cleanup paths.
func tryRemoveStaleLock(projectRoot string) (bool, error) {
	lockDir := LockPath(projectRoot)
	info, err := os.Stat(lockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	if !info.IsDir() {
		data := readFileOrEmpty(lockDir)
		doc, readErr := parseLockDocument(data)
		if readErr != nil || !lockHolderAlive(doc) {
			_ = os.Remove(lockDir)
			return true, nil
		}
		return false, nil
	}
	ownerPath := filepath.Join(lockDir, lockOwnerFile)
	data, readErr := os.ReadFile(ownerPath)
	dirMod := info.ModTime()
	if fi, statErr := os.Stat(ownerPath); statErr == nil {
		dirMod = fi.ModTime()
	}
	stale := func(owner []byte, mod time.Time) bool {
		if len(owner) > 0 {
			parsed, err := parseLockDocument(owner)
			if err == nil {
				return !lockHolderAlive(parsed)
			}
			if time.Since(mod) < lockGracePeriod {
				return false
			}
			return true
		}
		if time.Since(mod) < lockGracePeriod {
			return false
		}
		return true
	}
	tombstoneRoot := fsx.TombstoneRoot(lockDir)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			if time.Since(info.ModTime()) < lockGracePeriod {
				return false, nil
			}
			obs := fsx.LockObservation{DirMod: info.ModTime(), OwnerMissing: true}
			if err := fsx.ForceRemoveStaleDirLock(lockDir, obs, tombstoneRoot); err != nil {
				if errors.Is(err, os.ErrExist) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}
		if time.Since(dirMod) < lockGracePeriod {
			return false, nil
		}
		if stale(nil, dirMod) {
			obs := fsx.LockObservation{DirMod: dirMod, OwnerMissing: true}
			if err := fsx.ForceRemoveStaleDirLock(lockDir, obs, tombstoneRoot); err != nil {
				if errors.Is(err, os.ErrExist) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}
		return false, nil
	}
	if !stale(data, dirMod) {
		return false, nil
	}
	obs := fsx.ObservationFromOwner(data, dirMod, false)
	if err := fsx.ForceRemoveStaleDirLock(lockDir, obs, tombstoneRoot); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func readFileOrEmpty(path string) []byte {
	data, _ := os.ReadFile(path)
	return data
}
