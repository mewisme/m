package planner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/mewisme/mew/internal/linker"
)

// Capabilities describes filesystem link support between two roots.
type Capabilities struct {
	SameVolume bool `json:"sameVolume"`
	Hardlink   bool `json:"hardlink"`
	Reflink    bool `json:"reflink"`
	Symlink    bool `json:"symlink"`
	Junction   bool `json:"junction"`
}

var probeCache sync.Map // key: srcRoot+"\x00"+destRoot

// Probe detects link capabilities between srcRoot and destRoot.
func Probe(srcRoot, destRoot string) (Capabilities, error) {
	key := srcRoot + "\x00" + destRoot
	if v, ok := probeCache.Load(key); ok {
		return v.(Capabilities), nil
	}
	caps := probeFS(srcRoot, destRoot)
	probeCache.Store(key, caps)
	return caps, nil
}

// ProbeCached loads capabilities from cacheDir/fs-probe.json when present.
func ProbeCached(cacheDir, srcRoot, destRoot string) (Capabilities, error) {
	if cacheDir != "" {
		if caps, ok := loadProbeCache(cacheDir, srcRoot, destRoot); ok {
			return caps, nil
		}
	}
	caps, err := Probe(srcRoot, destRoot)
	if cacheDir != "" {
		_ = saveProbeCache(cacheDir, srcRoot, destRoot, caps)
	}
	return caps, err
}

type probeCacheFile struct {
	Entries map[string]Capabilities `json:"entries"`
}

func probeCacheKey(src, dest string) string {
	return src + " -> " + dest
}

func loadProbeCache(cacheDir, src, dest string) (Capabilities, bool) {
	path := filepath.Join(cacheDir, "fs-probe.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Capabilities{}, false
	}
	var doc probeCacheFile
	if err := json.Unmarshal(raw, &doc); err != nil || doc.Entries == nil {
		return Capabilities{}, false
	}
	caps, ok := doc.Entries[probeCacheKey(src, dest)]
	return caps, ok
}

func saveProbeCache(cacheDir, src, dest string, caps Capabilities) error {
	path := filepath.Join(cacheDir, "fs-probe.json")
	var doc probeCacheFile
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &doc)
	}
	if doc.Entries == nil {
		doc.Entries = map[string]Capabilities{}
	}
	doc.Entries[probeCacheKey(src, dest)] = caps
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// PlanFile picks the fastest safe op for one file path.
func PlanFile(src, dest string, caps Capabilities) linker.Op {
	if !caps.SameVolume {
		return linker.Op{Kind: linker.OpCopy, Src: src, Dest: dest}
	}
	if caps.Reflink {
		return linker.Op{Kind: linker.OpReflink, Src: src, Dest: dest}
	}
	if caps.Hardlink {
		return linker.Op{Kind: linker.OpHardlink, Src: src, Dest: dest}
	}
	return linker.Op{Kind: linker.OpCopy, Src: src, Dest: dest}
}

// PlanPackageLink picks the fastest safe op for linking a package directory tree
// into writable project node_modules. Hardlink is disabled so project mutations
// cannot alter immutable store content; use reflink then copy only.
func PlanPackageLink(src, dest string, caps Capabilities) linker.Op {
	if !caps.SameVolume {
		return linker.Op{Kind: linker.OpCopy, Src: src, Dest: dest}
	}
	if caps.Reflink {
		return linker.Op{Kind: linker.OpReflink, Src: src, Dest: dest}
	}
	return linker.Op{Kind: linker.OpCopy, Src: src, Dest: dest}
}

// PlanDirAlias picks symlink, junction, or copy for a directory alias.
func PlanDirAlias(src, dest string, caps Capabilities) linker.Op {
	if caps.Junction {
		return linker.Op{Kind: linker.OpJunction, Src: src, Dest: dest}
	}
	if caps.Symlink {
		return linker.Op{Kind: linker.OpSymlink, Src: src, Dest: dest}
	}
	return linker.Op{Kind: linker.OpCopy, Src: src, Dest: dest}
}
