package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

const indexSchemaVersion = 1

// IndexEntry is metadata for one imported package.
type IndexEntry struct {
	Integrity  string    `json:"integrity"`
	SizeBytes  int64     `json:"sizeBytes"`
	ImportedAt time.Time `json:"importedAt"`
}

// Index is the optional store/index.json document.
type Index struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Packages      map[string]IndexEntry `json:"packages"`
}

var indexMu sync.Mutex

func (s *PackageStore) indexPath() string {
	return filepath.Join(s.Root, "index.json")
}

func (s *PackageStore) loadIndex() (*Index, error) {
	path := s.indexPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{SchemaVersion: indexSchemaVersion, Packages: map[string]IndexEntry{}}, nil
		}
		return nil, apperr.Wrap(apperr.Store, "store.index", path, err)
	}
	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, apperr.Wrap(apperr.Store, "store.index", path, err)
	}
	if idx.Packages == nil {
		idx.Packages = map[string]IndexEntry{}
	}
	if idx.SchemaVersion == 0 {
		idx.SchemaVersion = indexSchemaVersion
	}
	return &idx, nil
}

func (s *PackageStore) indexUpsert(key PackageKey, integrity string, size int64) error {
	indexMu.Lock()
	defer indexMu.Unlock()

	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	idx.Packages[key.String()] = IndexEntry{
		Integrity:  integrity,
		SizeBytes:  size,
		ImportedAt: time.Now().UTC(),
	}
	return s.writeIndex(idx)
}

func (s *PackageStore) writeIndex(idx *Index) error {
	if idx == nil {
		return apperr.New(apperr.Store, "store.index", "", "nil index")
	}
	raw, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Store, "store.index", "", err)
	}
	raw = append(raw, '\n')
	path := s.indexPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.Store, "store.index", path, err)
	}
	return writeAtomic(path, raw)
}

// ReadIndex loads index.json from root.
func ReadIndex(root string) (*Index, error) {
	return NewPackageStore(root).loadIndex()
}

// Status reports package count and total bytes from index + filesystem scan fallback.
func (s *PackageStore) Status() (count int, bytes int64, err error) {
	if s == nil || s.Root == "" {
		return 0, 0, apperr.New(apperr.Store, "store.status", "", "nil store")
	}
	idx, err := s.loadIndex()
	if err != nil {
		return 0, 0, err
	}
	for _, e := range idx.Packages {
		count++
		bytes += e.SizeBytes
	}
	if count > 0 {
		return count, bytes, nil
	}
	pkgRoot := filepath.Join(s.Root, "packages")
	_ = filepath.WalkDir(pkgRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d == nil || d.IsDir() {
			return walkErr
		}
		if filepath.Base(path) == "package.json" {
			count++
		}
		if info, err := d.Info(); err == nil {
			bytes += info.Size()
		}
		return nil
	})
	return count, bytes, nil
}

// ListPackageKeys returns keys present under packages/.
func (s *PackageStore) ListPackageKeys() ([]PackageKey, error) {
	if s == nil || s.Root == "" {
		return nil, apperr.New(apperr.Store, "store.list", "", "nil store")
	}
	var keys []PackageKey
	pkgRoot := filepath.Join(s.Root, "packages")
	err := filepath.WalkDir(pkgRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil || !d.IsDir() || filepath.Base(path) != "package.json" {
			return nil
		}
		rel, err := filepath.Rel(pkgRoot, filepath.Dir(path))
		if err != nil {
			return err
		}
		algo, hex, ok := splitKeyPath(rel)
		if !ok {
			return nil
		}
		keys = append(keys, PackageKey{Algo: algo, Hex: hex})
		return nil
	})
	return keys, err
}

func splitKeyPath(rel string) (algo, hex string, ok bool) {
	rel = filepath.ToSlash(rel)
	algo, hex, ok = strings.Cut(rel, "/")
	if !ok || algo == "" || hex == "" {
		return "", "", false
	}
	return algo, hex, true
}
