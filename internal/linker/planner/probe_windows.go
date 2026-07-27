//go:build windows

package planner

import (
	"os"
	"path/filepath"
	"strings"
)

func platformSameVolume(a, b string) bool {
	volA := volumeName(a)
	volB := volumeName(b)
	return volA != "" && volA == volB
}

func volumeName(path string) string {
	path = filepath.Clean(path)
	if len(path) < 3 {
		return ""
	}
	return strings.ToUpper(path[:3])
}

func platformProbeReflink(_, _ string) bool {
	return false
}

func platformProbeJunction(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	target := filepath.Join(dir, ".mew-probe-junc-target")
	link := filepath.Join(dir, ".mew-probe-junc-link")
	_ = os.RemoveAll(link)
	_ = os.RemoveAll(target)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return false
	}
	defer func() {
		_ = os.RemoveAll(link)
		_ = os.RemoveAll(target)
	}()
	return platformJunction(target, link) == nil
}
