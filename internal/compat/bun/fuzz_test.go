package bun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func FuzzDecodeBunLock(f *testing.F) {
	root := testkit.ModuleRoot(f)
	seed := filepath.Join(root, "fixtures", "locks", "bun", "v1-basic", "bun.lock")
	if data, err := os.ReadFile(seed); err == nil {
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxLockBytes {
			return
		}
		_, _ = Decode(data)
	})
}
