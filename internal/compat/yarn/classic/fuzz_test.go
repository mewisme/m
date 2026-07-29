package classic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func FuzzDecodeYarnClassicLock(f *testing.F) {
	root := testkit.ModuleRoot(f)
	seed := filepath.Join(root, "fixtures", "locks", "yarn", "classic-v1", "yarn.lock")
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
