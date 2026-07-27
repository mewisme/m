package fsx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// OwnerFileName is the metadata file inside a directory lock.
	OwnerFileName = "owner.json"
	// DefaultLockGrace is how long a malformed owner file is tolerated after creation.
	DefaultLockGrace  = 5 * time.Second
	staleTombstoneDir = ".stale"
)

// ReleaseResult classifies directory lock release outcomes.
type ReleaseResult int

const (
	ReleaseOK ReleaseResult = iota
	ReleaseNotOwner
	ReleaseMissingOwner
	ReleaseMalformedOwner
)

// LockObservation captures lock metadata at stale-detection time for ABA-safe takeover.
type LockObservation struct {
	LockID        string
	PackageKey    string
	TxnID         string
	PID           int
	ProcessStart  int64
	SchemaVersion int
	ProjectRoot   string
	OwnerJSON     []byte
	DirMod        time.Time
	OwnerMissing  bool
}

// DirLockOptions tunes directory lock acquisition.
type DirLockOptions struct {
	RetryInterval time.Duration
	MaxWait       time.Duration
	GracePeriod   time.Duration
}

func (o DirLockOptions) withDefaults() DirLockOptions {
	if o.RetryInterval <= 0 {
		o.RetryInterval = 25 * time.Millisecond
	}
	if o.MaxWait <= 0 {
		o.MaxWait = 30 * time.Second
	}
	if o.GracePeriod <= 0 {
		o.GracePeriod = DefaultLockGrace
	}
	return o
}

// TombstoneRoot returns the sibling tombstone directory for lockDir.
func TombstoneRoot(lockDir string) string {
	return filepath.Join(filepath.Dir(lockDir), ".lock-tombstones")
}

// NewLockID returns a random hex identifier for lock ownership.
func NewLockID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// AcquireDirLock creates lockDir exclusively and writes ownerJSON inside it.
// stale reports whether an existing lock may be removed. matchOwner verifies release ownership.
func AcquireDirLock(
	ctx context.Context,
	lockDir string,
	ownerJSON []byte,
	opts DirLockOptions,
	stale func(ownerJSON []byte, dirMod time.Time) bool,
	matchOwner func(ownerJSON []byte) bool,
) (release func(), err error) {
	opts = opts.withDefaults()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := os.MkdirAll(filepath.Dir(lockDir), 0o755); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(opts.MaxWait)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	tombstoneRoot := TombstoneRoot(lockDir)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if removed, remErr := tryRemoveStaleDirLock(lockDir, opts.GracePeriod, stale, tombstoneRoot); remErr != nil {
			return nil, remErr
		} else if removed {
			if rel, acqErr := createDirLock(lockDir, ownerJSON, matchOwner); acqErr == nil {
				return rel, nil
			} else if !errors.Is(acqErr, os.ErrExist) {
				return nil, acqErr
			}
		} else if rel, acqErr := createDirLock(lockDir, ownerJSON, matchOwner); acqErr == nil {
			return rel, nil
		} else if !errors.Is(acqErr, os.ErrExist) {
			return nil, acqErr
		}
		if time.Now().After(deadline) {
			return nil, os.ErrExist
		}
		timer := time.NewTimer(opts.RetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func createDirLock(lockDir string, ownerJSON []byte, matchOwner func(ownerJSON []byte) bool) (func(), error) {
	if err := os.Mkdir(lockDir, 0o755); err != nil {
		return nil, err
	}
	ownerPath := filepath.Join(lockDir, OwnerFileName)
	if err := PublishFileDurable(ownerPath, ownerJSON, 0o644); err != nil {
		_ = os.RemoveAll(lockDir)
		return nil, err
	}
	return func() {
		_, _ = ReleaseDirLock(lockDir, matchOwner)
	}, nil
}

// ReleaseDirLock removes lockDir when matchOwner accepts the on-disk owner file.
func ReleaseDirLock(lockDir string, matchOwnerFn func(ownerJSON []byte) bool) (ReleaseResult, error) {
	ownerPath := filepath.Join(lockDir, OwnerFileName)
	data, err := os.ReadFile(ownerPath)
	if err != nil {
		if os.IsNotExist(err) {
			if _, statErr := os.Stat(lockDir); os.IsNotExist(statErr) {
				return ReleaseOK, nil
			}
			return ReleaseMissingOwner, nil
		}
		return ReleaseOK, err
	}
	if matchOwnerFn != nil {
		switch ownerMatchResult(data, matchOwnerFn) {
		case ownerMatchOK:
		case ownerMatchMalformed:
			return ReleaseMalformedOwner, nil
		default:
			return ReleaseNotOwner, nil
		}
	}
	if err := os.RemoveAll(lockDir); err != nil && !os.IsNotExist(err) {
		return ReleaseOK, err
	}
	return ReleaseOK, nil
}

type ownerMatchOutcome int

const (
	ownerMatchOK ownerMatchOutcome = iota
	ownerMatchNotOwner
	ownerMatchMalformed
)

func ownerMatchResult(data []byte, matchOwnerFn func([]byte) bool) ownerMatchOutcome {
	type ownerProbe struct {
		LockID string `json:"lockId"`
	}
	if len(data) > 0 {
		var probe ownerProbe
		if err := json.Unmarshal(data, &probe); err != nil {
			return ownerMatchMalformed
		}
	}
	if matchOwnerFn(data) {
		return ownerMatchOK
	}
	return ownerMatchNotOwner
}

// TakeoverStaleDirLock renames a stale lock directory into tombstoneRoot without deleting a recreated lock.
func TakeoverStaleDirLock(lockDir string, observed LockObservation, tombstoneRoot string) error {
	release, err := AcquireTakeoverGuard(context.Background(), lockDir)
	if err != nil {
		return err
	}
	defer release()

	current, err := ObserveDirLock(lockDir)
	if err != nil {
		if os.IsNotExist(err) {
			if observed.LockID != "" || len(observed.OwnerJSON) > 0 || !observed.OwnerMissing {
				return os.ErrExist
			}
			return nil
		}
		return err
	}
	if !ObservationMatches(observed, current) {
		return os.ErrExist
	}
	observed = current

	if observed.OwnerMissing {
		return tombstoneLockDir(lockDir, observed, tombstoneRoot)
	}
	ownerPath := filepath.Join(lockDir, OwnerFileName)
	data, err := os.ReadFile(ownerPath)
	if err != nil {
		if os.IsNotExist(err) {
			if _, statErr := os.Stat(lockDir); os.IsNotExist(statErr) {
				return nil
			}
			observed.OwnerMissing = true
			return tombstoneLockDir(lockDir, observed, tombstoneRoot)
		}
		return err
	}
	if observed.LockID != "" {
		gotID, err := parseLockID(data)
		if err != nil {
			return err
		}
		if gotID != observed.LockID {
			return os.ErrExist
		}
	}
	if len(observed.OwnerJSON) > 0 && string(data) != string(observed.OwnerJSON) {
		return os.ErrExist
	}
	return tombstoneLockDir(lockDir, observed, tombstoneRoot)
}

// ForceRemoveStaleDirLock tombstones a stale lock for recovery paths only.
func ForceRemoveStaleDirLock(lockDir string, observed LockObservation, tombstoneRoot string) error {
	if _, err := os.Stat(lockDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return TakeoverStaleDirLock(lockDir, observed, tombstoneRoot)
}

func tombstoneLockDir(lockDir string, observed LockObservation, tombstoneRoot string) error {
	staleDir := filepath.Join(tombstoneRoot, staleTombstoneDir)
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		return err
	}
	name := observed.LockID
	if name == "" {
		name = "missing"
	}
	tombPath := filepath.Join(staleDir, fmt.Sprintf("%s-%d", name, time.Now().UnixNano()))
	waitTakeoverPause("pre-rename")
	current, err := ObserveDirLock(lockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !ObservationMatches(observed, current) {
		return os.ErrExist
	}
	if err := os.Rename(lockDir, tombPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if observed.LockID != "" {
		data, err := os.ReadFile(filepath.Join(tombPath, OwnerFileName))
		if err != nil {
			return err
		}
		gotID, err := parseLockID(data)
		if err != nil {
			return err
		}
		if gotID != observed.LockID {
			return os.ErrExist
		}
	}
	CleanupTombstones(tombstoneRoot)
	return nil
}

// CleanupTombstones best-effort removes tombstoned lock directories.
func CleanupTombstones(tombstoneRoot string) {
	staleDir := filepath.Join(tombstoneRoot, staleTombstoneDir)
	entries, err := os.ReadDir(staleDir)
	if err != nil {
		return
	}
	for _, ent := range entries {
		_ = os.RemoveAll(filepath.Join(staleDir, ent.Name()))
	}
}

func tryRemoveStaleDirLock(lockDir string, grace time.Duration, stale func(ownerJSON []byte, dirMod time.Time) bool, tombstoneRoot string) (bool, error) {
	info, err := os.Stat(lockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	// Legacy file lock at the same path.
	if !info.IsDir() {
		if stale != nil && stale(readLegacyOwner(lockDir), info.ModTime()) {
			owner := readLegacyOwner(lockDir)
			obs := observationFromOwner(owner, info.ModTime(), false)
			if err := TakeoverStaleFileLock(lockDir, obs, tombstoneRoot); err != nil {
				if errors.Is(err, os.ErrExist) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}
		return false, nil
	}
	ownerPath := filepath.Join(lockDir, OwnerFileName)
	data, readErr := os.ReadFile(ownerPath)
	dirMod := info.ModTime()
	if fi, statErr := os.Stat(ownerPath); statErr == nil {
		dirMod = fi.ModTime()
	}
	if readErr != nil {
		if os.IsNotExist(readErr) {
			if time.Since(info.ModTime()) < grace {
				return false, nil
			}
			obs := LockObservation{DirMod: info.ModTime(), OwnerMissing: true}
			if err := ForceRemoveStaleDirLock(lockDir, obs, tombstoneRoot); err != nil {
				if errors.Is(err, os.ErrExist) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}
		if time.Since(dirMod) < grace {
			return false, nil
		}
		if stale != nil && stale(nil, dirMod) {
			obs := LockObservation{DirMod: dirMod, OwnerMissing: true}
			if err := ForceRemoveStaleDirLock(lockDir, obs, tombstoneRoot); err != nil {
				if errors.Is(err, os.ErrExist) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}
		return false, nil
	}
	if stale != nil && stale(data, dirMod) {
		obs := observationFromOwner(data, dirMod, false)
		if err := ForceRemoveStaleDirLock(lockDir, obs, tombstoneRoot); err != nil {
			if errors.Is(err, os.ErrExist) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func parseLockID(data []byte) (string, error) {
	type ownerID struct {
		LockID string `json:"lockId"`
	}
	var doc ownerID
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	return doc.LockID, nil
}

func readLegacyOwner(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

// ParseOwnerJSON unmarshals owner metadata.
func ParseOwnerJSON[T any](data []byte) (T, error) {
	var out T
	if len(data) == 0 {
		return out, errors.New("fsx.lockdir: empty owner")
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}
