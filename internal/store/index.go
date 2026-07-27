package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/diagnostics"
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

// ReconcileIndexResult reports index repair actions.
type ReconcileIndexResult struct {
	Added   int
	Removed int
}

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
		return nil, apperr.New(apperr.Store, "store.index", path, "missing schemaVersion")
	}
	if idx.SchemaVersion != indexSchemaVersion {
		return nil, apperr.New(apperr.Store, "store.index", path,
			fmt.Sprintf("unsupported schemaVersion %d", idx.SchemaVersion))
	}
	return &idx, nil
}

func (s *PackageStore) indexUpsert(key PackageKey, integrity string, size int64) error {
	release, err := acquireIndexLock(context.Background(), s.Root)
	if err != nil {
		return err
	}
	defer release()

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

func (s *PackageStore) indexUpsertOrWarn(key PackageKey, integrity string, size int64) {
	if err := s.indexUpsert(key, integrity, size); err != nil {
		s.warnIndex("store index upsert failed", key, err)
	}
}

func (s *PackageStore) warnIndex(msg string, key PackageKey, err error) {
	if s == nil || s.Reporter == nil || err == nil {
		return
	}
	line := "warning: " + msg + " key=" + key.String() + " error=" + err.Error()
	s.Reporter.Progress(diagnostics.Event{Phase: line})
}

func (s *PackageStore) writeIndex(idx *Index) error {
	if idx == nil {
		return apperr.New(apperr.Store, "store.index", "", "nil index")
	}
	if idx.SchemaVersion == 0 {
		idx.SchemaVersion = indexSchemaVersion
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

// ReconcileIndex rebuilds index.json from packages/ on disk.
// The index is a rebuildable cache; import and verify do not depend on it.
func (s *PackageStore) ReconcileIndex() (ReconcileIndexResult, error) {
	if s == nil || s.Root == "" {
		return ReconcileIndexResult{}, apperr.New(apperr.Store, "store.index", "", "nil store")
	}
	keys, err := s.ListPackageKeys()
	if err != nil {
		return ReconcileIndexResult{}, err
	}

	release, err := acquireIndexLock(context.Background(), s.Root)
	if err != nil {
		return ReconcileIndexResult{}, err
	}
	defer release()

	idx, err := s.loadIndex()
	if err != nil {
		return ReconcileIndexResult{}, err
	}

	desired := make(map[string]IndexEntry, len(keys))
	for _, key := range keys {
		entry, err := indexEntryFromPackage(s.PackagePath(key))
		if err != nil {
			return ReconcileIndexResult{}, apperr.Wrap(apperr.Store, "store.index", key.String(), err)
		}
		desired[key.String()] = entry
	}

	var result ReconcileIndexResult
	for k, entry := range desired {
		if old, ok := idx.Packages[k]; !ok || old.Integrity != entry.Integrity || old.SizeBytes != entry.SizeBytes {
			result.Added++
		}
	}
	for k := range idx.Packages {
		if _, ok := desired[k]; !ok {
			result.Removed++
		}
	}

	idx.Packages = desired
	if err := s.writeIndex(idx); err != nil {
		return ReconcileIndexResult{}, err
	}
	return result, nil
}

func indexEntryFromPackage(dir string) (IndexEntry, error) {
	markerPath := filepath.Join(dir, packageMarker)
	raw, err := os.ReadFile(markerPath)
	if err != nil {
		return IndexEntry{}, err
	}
	integrity := strings.TrimSpace(string(raw))
	size, err := dirSize(dir)
	if err != nil {
		return IndexEntry{}, err
	}
	return IndexEntry{
		Integrity:  integrity,
		SizeBytes:  size,
		ImportedAt: time.Now().UTC(),
	}, nil
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
	_, _ = s.CleanupStaleStaging(time.Hour)

	keys, err := s.ListPackageKeys()
	if err != nil {
		return 0, 0, err
	}
	fsCount := len(keys)

	idx, err := s.loadIndex()
	if err != nil {
		return 0, 0, err
	}
	if len(idx.Packages) != fsCount {
		if _, err := s.ReconcileIndex(); err != nil {
			return 0, 0, err
		}
		idx, err = s.loadIndex()
		if err != nil {
			return 0, 0, err
		}
	}

	if len(idx.Packages) > 0 {
		for _, e := range idx.Packages {
			count++
			bytes += e.SizeBytes
		}
		return count, bytes, nil
	}

	for _, key := range keys {
		entry, err := indexEntryFromPackage(s.PackagePath(key))
		if err != nil {
			return 0, 0, apperr.Wrap(apperr.Store, "store.status", key.String(), err)
		}
		count++
		bytes += entry.SizeBytes
	}
	return count, bytes, nil
}

// ListPackageKeys returns keys present under packages/.
func (s *PackageStore) ListPackageKeys() ([]PackageKey, error) {
	if s == nil || s.Root == "" {
		return nil, apperr.New(apperr.Store, "store.list", "", "nil store")
	}
	var keys []PackageKey
	pkgRoot := filepath.Join(s.Root, "packages")
	if _, err := os.Stat(pkgRoot); os.IsNotExist(err) {
		return nil, nil
	}
	err := filepath.WalkDir(pkgRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil || d.IsDir() || filepath.Base(path) != "package.json" {
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
