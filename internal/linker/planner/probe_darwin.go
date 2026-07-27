//go:build darwin

package planner

import (
	"os"
	"path/filepath"
	"syscall"
)

func platformSameVolume(a, b string) bool {
	var sa, sb syscall.Stat_t
	if err := syscall.Stat(a, &sa); err != nil {
		return false
	}
	if err := syscall.Stat(b, &sb); err != nil {
		return false
	}
	return sa.Dev == sb.Dev
}

func platformProbeReflink(srcRoot, destRoot string) bool {
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return false
	}
	src := filepath.Join(srcRoot, ".mew-probe-ref-src")
	dest := filepath.Join(destRoot, ".mew-probe-ref-dest")
	_ = os.Remove(src)
	_ = os.Remove(dest)
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		return false
	}
	defer func() {
		_ = os.Remove(src)
		_ = os.Remove(dest)
	}()
	if err := platformReflink(src, dest); err != nil {
		return false
	}
	return true
}

func platformProbeJunction(_ string) bool {
	return false
}
