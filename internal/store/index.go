package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
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

func (s *PackageStore) indexUpsert(key PackageKey, integrity string, size int64) (codes []string, warnings []string, err error) {
	release, err := acquireIndexLock(context.Background(), s.Root)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		c, w := s.releaseIndexLockWarn(release)
		codes = append(codes, c...)
		warnings = append(warnings, w...)
	}()

	idx, err := s.loadIndex()
	if err != nil {
		return codes, warnings, err
	}
	idx.Packages[key.String()] = IndexEntry{
		Integrity:  integrity,
		SizeBytes:  size,
		ImportedAt: time.Now().UTC(),
	}
	return codes, warnings, s.writeIndex(idx)
}

func (s *PackageStore) indexUpsertOrWarn(key PackageKey, integrity string, size int64) (codes []string, warnings []string) {
	codes, warnings, err := s.indexUpsert(key, integrity, size)
	if err != nil {
		s.warnIndex("store index upsert failed", key, err)
	}
	return codes, warnings
}

func (s *PackageStore) warnIndex(msg string, key PackageKey, err error) {
	if s == nil || s.Reporter == nil || err == nil {
		return
	}
	line := "warning: " + msg + " key=" + key.String() + " error=" + err.Error()
	s.Reporter.Progress(diagnostics.Event{Phase: line})
}

func (s *PackageStore) warnMaintenance(msg string, err error) {
	if s == nil || s.Reporter == nil || err == nil {
		return
	}
	line := "warning: " + msg + " error=" + err.Error()
	s.Reporter.Progress(diagnostics.Event{Phase: line})
}

func (s *PackageStore) releaseIndexLockWarn(release func() error) (codes []string, warnings []string) {
	if release == nil {
		return nil, nil
	}
	if err := release(); err != nil {
		s.warnMaintenance("store index lock release failed", err)
		return []string{CleanupCodeIndexLockRelease}, []string{err.Error()}
	}
	return nil, nil
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

func quarantineCorruptIndex(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	ts := time.Now().UTC().Format("20060102T150405.000000000Z")
	quarantine := path + ".corrupt." + ts
	return os.Rename(path, quarantine)
}

func indexFromFilesystem(s *PackageStore) (map[string]IndexEntry, error) {
	keys, err := s.ListPackageKeys()
	if err != nil {
		return nil, err
	}
	desired := make(map[string]IndexEntry, len(keys))
	for _, key := range keys {
		entry, err := indexEntryFromPackage(s.PackagePath(key))
		if err != nil {
			return nil, apperr.Wrap(apperr.Store, "store.index", key.String(), err)
		}
		desired[key.String()] = entry
	}
	return desired, nil
}

func indexMatches(idx *Index, desired map[string]IndexEntry) bool {
	if idx == nil || len(idx.Packages) != len(desired) {
		return false
	}
	for k, want := range desired {
		got, ok := idx.Packages[k]
		if !ok || got.Integrity != want.Integrity || got.SizeBytes != want.SizeBytes {
			return false
		}
	}
	return true
}

func reconcileIndexLocked(idx *Index, desired map[string]IndexEntry) (ReconcileIndexResult, Index, error) {
	if idx == nil {
		idx = &Index{SchemaVersion: indexSchemaVersion, Packages: map[string]IndexEntry{}}
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
	out := Index{SchemaVersion: indexSchemaVersion, Packages: desired}
	return result, out, nil
}

// ReconcileIndex rebuilds index.json from packages/ on disk.
// The index is a rebuildable cache; import and verify do not depend on it.
func (s *PackageStore) ReconcileIndex() (ReconcileIndexResult, error) {
	if s == nil || s.Root == "" {
		return ReconcileIndexResult{}, apperr.New(apperr.Store, "store.index", "", "nil store")
	}

	release, err := acquireIndexLock(context.Background(), s.Root)
	if err != nil {
		return ReconcileIndexResult{}, err
	}
	defer s.releaseIndexLockWarn(release)

	desired, err := indexFromFilesystem(s)
	if err != nil {
		return ReconcileIndexResult{}, err
	}

	path := s.indexPath()
	idx, loadErr := s.loadIndex()
	if loadErr != nil {
		if qerr := quarantineCorruptIndex(path); qerr != nil {
			return ReconcileIndexResult{}, apperr.Wrap(apperr.Store, "store.index", path, qerr)
		}
		idx = &Index{SchemaVersion: indexSchemaVersion, Packages: map[string]IndexEntry{}}
	}

	result, out, err := reconcileIndexLocked(idx, desired)
	if err != nil {
		return ReconcileIndexResult{}, err
	}
	if err := s.writeIndex(&out); err != nil {
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

// Status reports package count and total bytes from authoritative filesystem scan.
func (s *PackageStore) Status() (count int, bytes int64, err error) {
	if s == nil || s.Root == "" {
		return 0, 0, apperr.New(apperr.Store, "store.status", "", "nil store")
	}
	_, _ = s.CleanupStaleStaging(time.Hour)

	release, err := acquireIndexLock(context.Background(), s.Root)
	if err != nil {
		return 0, 0, err
	}
	defer s.releaseIndexLockWarn(release)

	desired, err := indexFromFilesystem(s)
	if err != nil {
		return 0, 0, err
	}

	path := s.indexPath()
	idx, loadErr := s.loadIndex()
	if loadErr != nil || !indexMatches(idx, desired) {
		if loadErr != nil {
			if qerr := quarantineCorruptIndex(path); qerr != nil {
				return 0, 0, apperr.Wrap(apperr.Store, "store.status", path, qerr)
			}
		}
		_, out, err := reconcileIndexLocked(idx, desired)
		if err != nil {
			return 0, 0, err
		}
		if err := s.writeIndex(&out); err != nil {
			return 0, 0, err
		}
	}

	for _, e := range desired {
		count++
		bytes += e.SizeBytes
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
