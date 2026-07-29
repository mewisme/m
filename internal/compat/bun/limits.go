package bun

import (
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
)

// ponytail: O(n) scan caps; upgrade path is streaming JSON with early abort.
const (
	maxLockBytes     = 32 << 20
	maxMapEntries    = 100_000
	maxPackageKeyLen = 4096
	maxDepNameLen    = 256
)

func validateLockInput(data []byte) error {
	if len(data) == 0 {
		return apperr.New(apperr.Lockfile, "bun.decode", "bun.lock", "empty document")
	}
	if len(data) > maxLockBytes {
		return apperr.New(apperr.Lockfile, "bun.decode", "bun.lock",
			fmt.Sprintf("lockfile exceeds %d byte limit", maxLockBytes))
	}
	return nil
}

func validatePackageName(name string) error {
	if len(name) > maxDepNameLen {
		return apperr.New(apperr.Lockfile, "bun.identity", name,
			fmt.Sprintf("package name exceeds %d bytes", maxDepNameLen))
	}
	return nil
}
