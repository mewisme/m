package mlock

import (
	"fmt"

	"github.com/mewisme/m/internal/apperr"
)

// Migrate upgrades doc to the current lockfile version.
func Migrate(doc *Document) error {
	if doc == nil {
		return apperr.New(apperr.Lockfile, "mlock.migrate", "m.lock", "nil document")
	}
	if doc.LockfileVersion == 0 {
		doc.LockfileVersion = LockfileVersion
	}
	if doc.LockfileVersion != LockfileVersion {
		return apperr.New(apperr.Lockfile, "mlock.migrate", "m.lock",
			fmt.Sprintf("unsupported lockfileVersion %d", doc.LockfileVersion))
	}
	return nil
}
