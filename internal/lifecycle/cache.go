package lifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mewisme/m/internal/apperr"
)

const cachePolicyVersion = 1

// ponytail: prepare-only cache marker files; upgrade path is output-dir capture.
func cacheKey(script Script) string {
	h := sha256.New()
	_, _ = h.Write([]byte(script.Integrity))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(script.Name))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(runtime.GOOS))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(runtime.GOARCH))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(itoa(cachePolicyVersion)))
	return hex.EncodeToString(h.Sum(nil))
}

func cacheHit(dir string, script Script) (bool, error) {
	if dir == "" || script.Name != "prepare" {
		return false, nil
	}
	path := filepath.Join(dir, cacheKey(script))
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, apperr.Wrap(apperr.IO, "lifecycle.cache", path, err)
	}
	return true, nil
}

func markCache(dir string, script Script) error {
	if dir == "" || script.Name != "prepare" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "lifecycle.cache", dir, err)
	}
	path := filepath.Join(dir, cacheKey(script))
	return os.WriteFile(path, []byte("1\n"), 0o644)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
