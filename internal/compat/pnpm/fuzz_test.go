package pnpm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func FuzzDecodePnpmLock(f *testing.F) {
	root := testkit.ModuleRoot(f)
	seeds := []string{
		filepath.Join(root, "fixtures", "locks", "pnpm", "v9", "pnpm-lock.yaml"),
		filepath.Join(root, "fixtures", "locks", "pnpm", "v10", "pnpm-lock.yaml"),
		filepath.Join(root, "fixtures", "locks", "pnpm", "v11", "pnpm-lock.yaml"),
		filepath.Join(root, "testdata", "lockfile", "fuzz", "duplicate-key.yaml"),
		filepath.Join(root, "testdata", "lockfile", "fuzz", "oversize-marker.yaml"),
	}
	for _, seed := range seeds {
		if data, err := os.ReadFile(seed); err == nil {
			f.Add(data)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxLockBytes {
			return
		}
		_, _ = Decode(data)
	})
}
