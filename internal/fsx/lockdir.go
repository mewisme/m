package fsx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const (
	// OwnerFileName is the metadata file inside a directory lock.
	OwnerFileName = "owner.json"
	// DefaultLockGrace is how long a malformed owner file is tolerated after creation.
	DefaultLockGrace = 5 * time.Second
)

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
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if removed, remErr := tryRemoveStaleDirLock(lockDir, opts.GracePeriod, stale); remErr != nil {
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
	if err := WriteAtomic(ownerPath, ownerJSON, 0o644); err != nil {
		_ = os.RemoveAll(lockDir)
		return nil, err
	}
	return func() {
		_ = ReleaseDirLock(lockDir, matchOwner)
	}, nil
}

// ReleaseDirLock removes lockDir when matchOwner accepts the on-disk owner file.
func ReleaseDirLock(lockDir string, matchOwnerFn func(ownerJSON []byte) bool) error {
	ownerPath := filepath.Join(lockDir, OwnerFileName)
	data, err := os.ReadFile(ownerPath)
	if err != nil {
		if os.IsNotExist(err) {
			if _, statErr := os.Stat(lockDir); os.IsNotExist(statErr) {
				return nil
			}
		} else {
			return err
		}
	} else if matchOwnerFn != nil && !matchOwnerFn(data) {
		return nil
	}
	if err := os.RemoveAll(lockDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func tryRemoveStaleDirLock(lockDir string, grace time.Duration, stale func(ownerJSON []byte, dirMod time.Time) bool) (bool, error) {
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
			_ = os.Remove(lockDir)
			return true, nil
		}
		return false, nil
	}
	ownerPath := filepath.Join(lockDir, OwnerFileName)
	data, readErr := os.ReadFile(ownerPath)
	dirMod := info.ModTime()
	if fi, err := os.Stat(ownerPath); err == nil {
		dirMod = fi.ModTime()
	}
	if readErr != nil {
		if os.IsNotExist(readErr) {
			if time.Since(info.ModTime()) < grace {
				return false, nil
			}
			_ = os.RemoveAll(lockDir)
			return true, nil
		}
		if time.Since(dirMod) < grace {
			return false, nil
		}
		if stale != nil && stale(nil, dirMod) {
			_ = os.RemoveAll(lockDir)
			return true, nil
		}
		return false, nil
	}
	if stale != nil && stale(data, dirMod) {
		_ = os.RemoveAll(lockDir)
		return true, nil
	}
	return false, nil
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
