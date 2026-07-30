package dlx

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// LockDomain identifies a mx cache lock namespace.
type LockDomain string

const (
	LockConsent     LockDomain = "consent"
	LockRequest     LockDomain = "request"
	LockEnvironment LockDomain = "environment"
)

// LockPath returns the directory lock path for a domain and digest.
func LockPath(mxCacheDir string, domain LockDomain, digest string) string {
	return filepath.Join(mxCacheDir, "locks", string(domain), digest+".lock")
}

type lockOwner struct {
	SchemaVersion int    `json:"schemaVersion"`
	Domain        string `json:"domain"`
	Digest        string `json:"digest"`
	LockID        string `json:"lockId"`
}

// AcquireLock acquires a cross-process directory lock under mxCacheDir.
func AcquireLock(ctx context.Context, mxCacheDir string, domain LockDomain, digest string) (release func(), err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	lockID, err := fsx.NewLockID()
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "dlx.lock", string(domain), err)
	}
	owner := lockOwner{SchemaVersion: 1, Domain: string(domain), Digest: digest, LockID: lockID}
	ownerJSON, err := json.Marshal(owner)
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "dlx.lock", string(domain), err)
	}
	lockDir := LockPath(mxCacheDir, domain, digest)
	match := func(b []byte) bool {
		var o lockOwner
		if json.Unmarshal(b, &o) != nil {
			return false
		}
		return o.LockID == lockID
	}
	releaseFn, err := fsx.AcquireDirLock(ctx, lockDir, ownerJSON, fsx.DirLockOptions{}, nil, match)
	if err != nil {
		if ctx.Err() != nil {
			return nil, apperr.Wrap(apperr.Timeout, "dlx.lock", string(domain), err)
		}
		return nil, apperr.Wrap(apperr.IO, "dlx.lock", string(domain), err)
	}
	return releaseFn, nil
}

// ConsentLockPath returns the consent-store lock path.
func ConsentLockPath(mxCacheDir string) string {
	return LockPath(mxCacheDir, LockConsent, "store")
}

// EnsureLockParent creates parent directories for a lock path.
func EnsureLockParent(lockDir string) error {
	return os.MkdirAll(filepath.Dir(lockDir), 0o755)
}
