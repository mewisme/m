package fsx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TakeoverStaleFileLock tombstones a stale legacy file lock without os.Remove on the authoritative path.
func TakeoverStaleFileLock(lockPath string, observed LockObservation, tombstoneRoot string) error {
	release, err := AcquireTakeoverGuard(context.Background(), lockPath)
	if err != nil {
		return err
	}
	defer release()

	current, err := observeFileLock(lockPath)
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
	return tombstoneLockFile(lockPath, current, tombstoneRoot)
}

func observeFileLock(lockPath string) (LockObservation, error) {
	info, err := os.Stat(lockPath)
	if err != nil {
		return LockObservation{}, err
	}
	if info.IsDir() {
		return LockObservation{}, os.ErrExist
	}
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return LockObservation{}, err
	}
	return observationFromOwner(data, info.ModTime(), false), nil
}

func tombstoneLockFile(lockPath string, observed LockObservation, tombstoneRoot string) error {
	staleDir := filepath.Join(tombstoneRoot, staleTombstoneDir)
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		return err
	}
	name := observed.LockID
	if name == "" {
		name = "legacy-file"
	}
	tombPath := filepath.Join(staleDir, fmt.Sprintf("%s-%d", name, time.Now().UnixNano()))
	waitTakeoverPause("pre-rename")
	current, err := observeFileLock(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !ObservationMatches(observed, current) {
		return os.ErrExist
	}
	if err := os.Rename(lockPath, tombPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	CleanupTombstones(tombstoneRoot)
	return nil
}
