package npm

import (
	"fmt"

	"github.com/mewisme/mew/internal/lockfile"
)

// ValidateSupported rejects unsupported npm lockfile versions.
func ValidateSupported(doc *Document) error {
	if doc == nil {
		return lockfile.NewUnsupported("npm.validate", "package-lock.json", "nil document")
	}
	switch doc.LockfileVersion {
	case 2, 3:
		return nil
	case 0:
		return lockfile.NewUnsupported("npm.validate", "package-lock.json", "missing lockfileVersion")
	default:
		return lockfile.NewUnsupported("npm.validate", "package-lock.json",
			fmt.Sprintf("unsupported lockfileVersion %d (only 2 and 3 are supported)", doc.LockfileVersion))
	}
}
