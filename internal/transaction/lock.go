package transaction

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

const lockFileName = "lock"

// LockDocument is persisted at .mew/txn/lock.
type LockDocument struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
	TxnID     string    `json:"txnId"`
}

// LockPath returns <project>/.mew/txn/lock.
func LockPath(projectRoot string) string {
	return filepath.Join(TxnRoot(projectRoot), lockFileName)
}

// AcquireProjectLock creates the project lock or returns an error when held.
func AcquireProjectLock(projectRoot, txnID string) error {
	path := LockPath(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	if stale, err := isStaleLock(path); err != nil {
		return err
	} else if !stale {
		if _, err := os.Stat(path); err == nil {
			return apperr.New(apperr.Transaction, "transaction.lock", path,
				"another install transaction is in progress")
		}
	}
	doc := LockDocument{PID: os.Getpid(), StartedAt: time.Now().UTC(), TxnID: txnID}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Transaction, "transaction.lock", path, err)
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	return nil
}

// ReleaseProjectLock removes the project lock when owned by this process.
func ReleaseProjectLock(projectRoot string) error {
	path := LockPath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	var doc LockDocument
	if err := json.Unmarshal(data, &doc); err == nil && doc.PID != 0 && doc.PID != os.Getpid() {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	return nil
}

func isStaleLock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	var doc LockDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return true, nil
	}
	if doc.PID <= 0 {
		return true, nil
	}
	if processAlive(doc.PID) {
		return false, nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return false, apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	return true, nil
}
