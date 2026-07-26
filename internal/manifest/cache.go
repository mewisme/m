package manifest

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

type cacheEntry struct {
	modTime time.Time
	doc     *Document
}

var (
	docCache sync.Map // string abs dir -> cacheEntry
	cacheMu  sync.Mutex
)

// LoadCached loads dir/package.json, reusing a parse when mtime is unchanged.
func LoadCached(dir string) (*Document, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "manifest.cache", dir, err)
	}
	pkg := filepath.Join(abs, "package.json")
	st, err := os.Stat(pkg)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.Wrap(apperr.NotFound, "manifest.cache", pkg, err)
		}
		return nil, apperr.Wrap(apperr.IO, "manifest.cache", pkg, err)
	}
	if v, ok := docCache.Load(abs); ok {
		ent := v.(cacheEntry)
		if ent.modTime.Equal(st.ModTime()) && ent.doc != nil {
			return cloneDoc(ent.doc), nil
		}
	}
	doc, err := Load(pkg)
	if err != nil {
		return nil, err
	}
	docCache.Store(abs, cacheEntry{modTime: st.ModTime(), doc: cloneDoc(doc)})
	return doc, nil
}

// Invalidate drops the cached parse for root (watcher hook for later MVPs).
func Invalidate(root string) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return
	}
	docCache.Delete(abs)
}

func cloneDoc(d *Document) *Document {
	if d == nil {
		return nil
	}
	out := *d
	out.Source = append([]byte(nil), d.Source...)
	out.Dependencies = cloneMap(d.Dependencies)
	out.DevDependencies = cloneMap(d.DevDependencies)
	out.OptionalDependencies = cloneMap(d.OptionalDependencies)
	out.PeerDependencies = cloneMap(d.PeerDependencies)
	out.Scripts = cloneMap(d.Scripts)
	out.Engines = cloneMap(d.Engines)
	if d.Workspaces != nil {
		out.Workspaces = append(out.Workspaces[:0:0], d.Workspaces...)
	}
	if d.Bin != nil {
		out.Bin = append(out.Bin[:0:0], d.Bin...)
	}
	return &out
}

// ClearCacheForTest empties the manifest cache (tests only).
func ClearCacheForTest() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	docCache.Range(func(k, _ any) bool {
		docCache.Delete(k)
		return true
	})
}
