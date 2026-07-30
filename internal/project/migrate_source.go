package project

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// LockCandidate is a migratable incumbent lockfile on disk.
type LockCandidate struct {
	File string
	ID   Identity
}

var migratableLockPrecedence = []LockCandidate{
	{File: "nub.lock", ID: IdentityNub},
	{File: "pnpm-lock.yaml", ID: IdentityPNPM},
	{File: npmShrinkwrapJSON, ID: IdentityNPM},
	{File: packageLockJSON, ID: IdentityNPM},
	{File: "yarn.lock", ID: IdentityYarn},
	{File: "bun.lock", ID: IdentityBun},
}

// MigratableLockCandidates lists incumbent locks that can be migrated to m.lock.
// m.lock and bun.lockb are excluded. When both npm lock files exist, shrinkwrap wins.
func MigratableLockCandidates(root string) []LockCandidate {
	hasShrinkwrap := false
	if _, err := os.Stat(filepath.Join(root, npmShrinkwrapJSON)); err == nil {
		hasShrinkwrap = true
	}
	var out []LockCandidate
	for _, c := range migratableLockPrecedence {
		if c.File == packageLockJSON && hasShrinkwrap {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, c.File)); err == nil {
			out = append(out, c)
		}
	}
	return out
}

// DetectIdentityForMigrate loads manifest identity without lockfile conflict checks.
func DetectIdentityForMigrate(root string) (*Project, error) {
	pkgPath := filepath.Join(root, "package.json")
	fieldID, fieldSig, fieldKind, err := readPackageManagerField(pkgPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	p := &Project{Root: root}
	if fieldID != "" {
		p.Signals = append(p.Signals, Signal{Kind: fieldKind, Detail: fieldSig, Path: pkgPath})
		p.Identity = fieldID
		return p, nil
	}
	p.Identity = IdentityMew
	p.Signals = append(p.Signals, Signal{Kind: "default", Detail: "mew native", Path: root})
	return p, nil
}

// ResolveMigrateSource picks the incumbent lock to migrate when --from is omitted.
// When fromFlag is set, it must be nub, pnpm, npm, bun, or yarn.
func ResolveMigrateSource(root string, fromFlag string) (Identity, string, error) {
	if fromFlag != "" {
		fromID, err := parseMigrateFromFlag(fromFlag)
		if err != nil {
			return "", "", err
		}
		lockPath, ok := IncumbentLockPath(root, fromID)
		if !ok {
			return "", "", apperr.New(apperr.NotFound, "lock.migrate", LockFilename(fromID), "source lock not found")
		}
		return fromID, lockPath, nil
	}

	pkgPath := filepath.Join(root, "package.json")
	fieldID, _, _, err := readPackageManagerField(pkgPath)
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	if fieldID != "" {
		return resolveMigrateFromManifest(root, fieldID)
	}
	return resolveMigrateFromLocks(root)
}

func resolveMigrateFromManifest(root string, fieldID Identity) (Identity, string, error) {
	if fieldID == IdentityMew {
		migratable := MigratableLockCandidates(root)
		if len(migratable) == 0 {
			if _, ok := IncumbentLockPath(root, IdentityMew); ok {
				return "", "", apperr.New(apperr.Usage, "lock.migrate", "m.lock", "nothing to migrate; project already uses m.lock")
			}
			return "", "", apperr.New(apperr.Usage, "lock.migrate", root, "nothing to migrate")
		}
		return "", "", apperr.New(apperr.Usage, "lock.migrate", string(fieldID),
			"packageManager declares mew/m; pass --from nub|pnpm|npm|bun|yarn to select incumbent lock")
	}
	if !isMigratableIdentity(fieldID) {
		return "", "", apperr.New(apperr.Usage, "lock.migrate", string(fieldID),
			"project identity is not nub, pnpm, npm, bun, or yarn")
	}
	lockPath, ok := IncumbentLockPath(root, fieldID)
	if !ok {
		for _, c := range MigratableLockCandidates(root) {
			if c.ID != fieldID {
				return "", "", apperr.New(apperr.Config, "identity", root,
					"conflicting signals: package field is "+string(fieldID)+" but lockfile is "+string(c.ID))
			}
		}
		return "", "", apperr.New(apperr.NotFound, "lock.migrate", LockFilename(fieldID), "source lock not found")
	}
	return fieldID, lockPath, nil
}

func resolveMigrateFromLocks(root string) (Identity, string, error) {
	cands := MigratableLockCandidates(root)
	switch len(cands) {
	case 0:
		if _, ok := IncumbentLockPath(root, IdentityMew); ok {
			return "", "", apperr.New(apperr.Usage, "lock.migrate", "m.lock", "nothing to migrate; project already uses m.lock")
		}
		return "", "", apperr.New(apperr.Usage, "lock.migrate", root, "nothing to migrate")
	case 1:
		path := filepath.Join(root, cands[0].File)
		return cands[0].ID, path, nil
	default:
		names := make([]string, len(cands))
		for i, c := range cands {
			names[i] = c.File
		}
		msg := "multiple lockfiles present; pass --from nub|pnpm|npm|bun|yarn\nfound: " + strings.Join(names, ", ")
		return "", "", apperr.New(apperr.Usage, "lock.migrate", root, msg)
	}
}

func parseMigrateFromFlag(from string) (Identity, error) {
	switch from {
	case "nub":
		return IdentityNub, nil
	case "pnpm":
		return IdentityPNPM, nil
	case "npm":
		return IdentityNPM, nil
	case "bun":
		return IdentityBun, nil
	case "yarn":
		return IdentityYarn, nil
	default:
		return "", apperr.New(apperr.Usage, "lock.migrate", from, "expected --from nub, pnpm, npm, bun, or yarn")
	}
}

func isMigratableIdentity(id Identity) bool {
	switch id {
	case IdentityNub, IdentityPNPM, IdentityNPM, IdentityBun, IdentityYarn:
		return true
	default:
		return false
	}
}
