package npm

import (
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
)

// ponytail: O(n) scan caps; upgrade path is streaming JSON with early abort.
const (
	maxLockBytes     = 32 << 20 // 32 MiB
	maxMapEntries    = 100_000
	maxPackageKeyLen = 4096
	maxIndexKeys     = 200_000
	maxDepNameLen    = 256
)

func validateLockInput(data []byte) error {
	if len(data) == 0 {
		return apperr.New(apperr.Lockfile, "npm.decode", "package-lock.json", "empty document")
	}
	if len(data) > maxLockBytes {
		return apperr.New(apperr.Lockfile, "npm.decode", "package-lock.json",
			fmt.Sprintf("lockfile exceeds %d byte limit", maxLockBytes))
	}
	return nil
}

func validatePackagePath(path string) error {
	if len(path) > maxPackageKeyLen {
		return apperr.New(apperr.Lockfile, "npm.identity", path,
			fmt.Sprintf("package path exceeds %d bytes", maxPackageKeyLen))
	}
	return nil
}

func validateDepName(name string) error {
	if len(name) > maxDepNameLen {
		return apperr.New(apperr.Lockfile, "npm.identity", name,
			fmt.Sprintf("dependency name exceeds %d bytes", maxDepNameLen))
	}
	return nil
}
