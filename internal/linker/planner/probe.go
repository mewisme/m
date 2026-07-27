package planner

import (
	"os"
	"path/filepath"
)

func probeFS(srcRoot, destRoot string) Capabilities {
	caps := Capabilities{}
	srcRoot, _ = filepath.Abs(srcRoot)
	destRoot, _ = filepath.Abs(destRoot)
	caps.SameVolume = sameVolume(srcRoot, destRoot)
	caps.Symlink = probeSymlink(destRoot)
	caps.Junction = probeJunction(destRoot)
	if !caps.SameVolume {
		return caps
	}
	caps.Hardlink = probeHardlink(srcRoot, destRoot)
	caps.Reflink = probeReflink(srcRoot, destRoot)
	return caps
}

func probeSymlink(dir string) bool {
	target := filepath.Join(dir, ".mew-probe-symlink")
	link := target + "-link"
	_ = os.Remove(link)
	if err := os.Symlink(target, link); err != nil {
		return false
	}
	_ = os.Remove(link)
	return true
}

func probeHardlink(srcRoot, destRoot string) bool {
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return false
	}
	src := filepath.Join(srcRoot, ".mew-probe-hl-src")
	dest := filepath.Join(destRoot, ".mew-probe-hl-dest")
	_ = os.Remove(src)
	_ = os.Remove(dest)
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		return false
	}
	defer func() {
		_ = os.Remove(src)
		_ = os.Remove(dest)
	}()
	if err := os.Link(src, dest); err != nil {
		return false
	}
	return true
}

func probeReflink(srcRoot, destRoot string) bool {
	return platformProbeReflink(srcRoot, destRoot)
}

func probeJunction(dir string) bool {
	return platformProbeJunction(dir)
}

func sameVolume(a, b string) bool {
	return platformSameVolume(a, b)
}
