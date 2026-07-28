package project

import (
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
)

// LockFilename returns the incumbent lockfile basename for identity.
func LockFilename(id Identity) string {
	switch id {
	case IdentityMew:
		return "m.lock"
	case IdentityNub:
		return "nub.lock"
	case IdentityPNPM:
		return "pnpm-lock.yaml"
	case IdentityNPM:
		return "package-lock.json"
	case IdentityYarn:
		return "yarn.lock"
	case IdentityBun:
		return "bun.lock"
	default:
		return ""
	}
}

// IncumbentLockPath returns the absolute incumbent lock path when the file exists.
func IncumbentLockPath(root string, id Identity) (string, bool) {
	name := LockFilename(id)
	if name == "" {
		return "", false
	}
	path := filepath.Join(root, name)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// DetectIncumbentLock scans the project root for a single recognized lockfile.
func DetectIncumbentLock(root string) (Identity, string, bool) {
	found := listLockfiles(root)
	if len(found) != 1 {
		return "", "", false
	}
	path := filepath.Join(root, found[0].file)
	return found[0].id, path, true
}

// ReadLockfileBytes reads incumbent lock bytes for txn backup and byte preservation.
func ReadLockfileBytes(root string, id Identity) ([]byte, error) {
	path, ok := IncumbentLockPath(root, id)
	if !ok {
		return nil, apperr.New(apperr.NotFound, "project.lock.read", LockFilename(id), "lockfile not found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "project.lock.read", path, err)
	}
	return data, nil
}
