package fsx

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ownerIdentity struct {
	SchemaVersion int    `json:"schemaVersion"`
	LockID        string `json:"lockId"`
	PID           int    `json:"pid"`
	ProcessStart  int64  `json:"processStart"`
	TxnID         string `json:"txnId"`
	PackageKey    string `json:"packageKey"`
	ProjectRoot   string `json:"projectRoot"`
}

// TakeoverGuardPath returns the advisory-lock file for lockDir's takeover namespace.
func TakeoverGuardPath(lockDir string) string {
	base := filepath.Dir(lockDir)
	ns := takeoverNamespace(lockDir)
	return filepath.Join(base, ".takeover-guards", ns+".guard")
}

func takeoverNamespace(lockDir string) string {
	lockDir = filepath.Clean(lockDir)
	base := filepath.Base(lockDir)
	parent := filepath.Base(filepath.Dir(lockDir))
	if parent != "" && parent != "." {
		return sanitizeTakeoverName(parent + "-" + base)
	}
	return sanitizeTakeoverName(base)
}

func sanitizeTakeoverName(s string) string {
	s = strings.NewReplacer(string(filepath.Separator), "_", "..", "_", ":", "_").Replace(s)
	if s == "" {
		return "lock"
	}
	return s
}

// AcquireTakeoverGuard blocks until the per-namespace advisory lock is held.
func AcquireTakeoverGuard(ctx context.Context, lockDir string) (release func(), err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	path := TakeoverGuardPath(lockDir)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		release, err := tryAcquireTakeoverGuard(path)
		if err == nil {
			return release, nil
		}
		if !isTakeoverGuardBusy(err) {
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// ObserveDirLock reads the current lock directory into a LockObservation.
func ObserveDirLock(lockDir string) (LockObservation, error) {
	info, err := os.Stat(lockDir)
	if err != nil {
		return LockObservation{}, err
	}
	if !info.IsDir() {
		return LockObservation{DirMod: info.ModTime(), OwnerJSON: readLegacyOwner(lockDir)}, nil
	}
	ownerPath := filepath.Join(lockDir, OwnerFileName)
	data, readErr := os.ReadFile(ownerPath)
	dirMod := info.ModTime()
	if fi, statErr := os.Stat(ownerPath); statErr == nil {
		dirMod = fi.ModTime()
	}
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return LockObservation{DirMod: dirMod, OwnerMissing: true}, nil
		}
		return LockObservation{}, readErr
	}
	return observationFromOwner(data, dirMod, false), nil
}

// ObservationFromOwner builds a LockObservation from owner metadata bytes.
func ObservationFromOwner(data []byte, dirMod time.Time, ownerMissing bool) LockObservation {
	return observationFromOwner(data, dirMod, ownerMissing)
}

func observationFromOwner(data []byte, dirMod time.Time, ownerMissing bool) LockObservation {
	obs := LockObservation{DirMod: dirMod, OwnerMissing: ownerMissing}
	if ownerMissing || len(data) == 0 {
		return obs
	}
	obs.OwnerJSON = append([]byte(nil), data...)
	var id ownerIdentity
	if err := json.Unmarshal(data, &id); err == nil {
		obs.LockID = id.LockID
		obs.TxnID = id.TxnID
		obs.PackageKey = id.PackageKey
		obs.PID = id.PID
		obs.ProcessStart = id.ProcessStart
		obs.SchemaVersion = id.SchemaVersion
		obs.ProjectRoot = id.ProjectRoot
	} else if lockID, err := parseLockID(data); err == nil {
		obs.LockID = lockID
	}
	return obs
}

// ObservationMatches reports whether current still matches the earlier observation.
func ObservationMatches(observed, current LockObservation) bool {
	if observed.OwnerMissing != current.OwnerMissing {
		return false
	}
	if observed.LockID != "" && current.LockID != observed.LockID {
		return false
	}
	if len(observed.OwnerJSON) > 0 && string(current.OwnerJSON) != string(observed.OwnerJSON) {
		return false
	}
	if observed.TxnID != "" && current.TxnID != observed.TxnID {
		return false
	}
	if observed.PackageKey != "" && current.PackageKey != observed.PackageKey {
		return false
	}
	if observed.ProjectRoot != "" && current.ProjectRoot != observed.ProjectRoot {
		return false
	}
	if observed.SchemaVersion != 0 && current.SchemaVersion != observed.SchemaVersion {
		return false
	}
	if observed.PID != 0 && current.PID != observed.PID {
		return false
	}
	if observed.ProcessStart != 0 && current.ProcessStart != observed.ProcessStart {
		return false
	}
	return true
}

func waitTakeoverPause(phase string) {
	want := strings.TrimSpace(os.Getenv("MEW_LOCK_TAKEOVER_PAUSE"))
	if want != phase {
		return
	}
	if ready := strings.TrimSpace(os.Getenv("MEW_LOCK_TAKEOVER_READY")); ready != "" {
		_ = os.WriteFile(ready, []byte("paused"), 0o644)
	}
	signal := strings.TrimSpace(os.Getenv("MEW_LOCK_TAKEOVER_SIGNAL"))
	if signal == "" {
		time.Sleep(30 * time.Second)
		return
	}
	for i := 0; i < 3000; i++ {
		if _, err := os.Stat(signal); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
