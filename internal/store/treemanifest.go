package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/mewisme/m/internal/apperr"
)

const (
	treeManifestSchemaVersion = 2
	treeManifestFileName      = ".mew-tree-manifest.json"
)

// TreeEntryKind classifies one manifest row.
type TreeEntryKind string

const (
	EntryFile    TreeEntryKind = "file"
	EntrySymlink TreeEntryKind = "symlink"
	EntryDir     TreeEntryKind = "dir"
)

// TreeEntry is one path in a published package tree.
type TreeEntry struct {
	Path          string `json:"path"`
	Kind          string `json:"kind"`
	Hash          string `json:"hash,omitempty"`
	Mode          uint32 `json:"mode"`
	SymlinkTarget string `json:"symlinkTarget,omitempty"`
}

// TreeManifest is the immutable content index for one store package.
type TreeManifest struct {
	SchemaVersion int         `json:"schemaVersion"`
	Entries       []TreeEntry `json:"entries"`
}

func treeManifestPath(dir string) string {
	return filepath.Join(dir, treeManifestFileName)
}

func generateTreeManifest(root string) (*TreeManifest, error) {
	var entries []TreeEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		base := filepath.Base(rel)
		if base == treeManifestFileName || base == packageMarker {
			return nil
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := uint32(info.Mode().Perm())
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entries = append(entries, TreeEntry{
				Path:          rel,
				Kind:          string(EntrySymlink),
				Mode:          mode,
				SymlinkTarget: filepath.ToSlash(target),
			})
			return nil
		}
		if d.IsDir() {
			entries = append(entries, TreeEntry{
				Path: rel,
				Kind: string(EntryDir),
				Mode: mode,
			})
			return nil
		}
		hash, err := hashFile(path)
		if err != nil {
			return err
		}
		entries = append(entries, TreeEntry{
			Path: rel,
			Kind: string(EntryFile),
			Hash: hash,
			Mode: mode,
		})
		return nil
	})
	if err != nil {
		return nil, apperr.Wrap(apperr.Store, "store.manifest", root, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return &TreeManifest{SchemaVersion: treeManifestSchemaVersion, Entries: entries}, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeTreeManifest(dir string, m *TreeManifest) error {
	if m == nil {
		return apperr.New(apperr.Store, "store.manifest", dir, "nil manifest")
	}
	if err := validateTreeManifest(m); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Store, "store.manifest", dir, err)
	}
	raw = append(raw, '\n')
	return os.WriteFile(treeManifestPath(dir), raw, 0o444)
}

func readTreeManifest(dir string) (*TreeManifest, error) {
	raw, err := os.ReadFile(treeManifestPath(dir))
	if err != nil {
		return nil, err
	}
	var m TreeManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, apperr.Wrap(apperr.Store, "store.manifest", dir, err)
	}
	if m.SchemaVersion == 0 {
		return nil, apperr.New(apperr.Store, "store.manifest", dir, "missing schemaVersion")
	}
	if err := validateTreeManifest(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// portableCollisionKey folds case, normalizes slashes, and trims Windows trailing
// dots/spaces per path segment so manifests stay safe on case-insensitive targets.
func portableCollisionKey(path string) string {
	path = filepath.ToSlash(path)
	segs := strings.Split(path, "/")
	out := make([]string, 0, len(segs))
	for _, seg := range segs {
		if seg == "" || seg == "." {
			continue
		}
		seg = strings.TrimRight(seg, ". ")
		seg = strings.ToLower(seg)
		out = append(out, seg)
	}
	return strings.Join(out, "/")
}

func portablePathCollisionErr(a, b string) error {
	return apperr.New(apperr.Store, "store.verify", a,
		fmt.Sprintf("portable path collision with %q", b))
}

func validateTreeManifest(m *TreeManifest) error {
	if m == nil {
		return apperr.New(apperr.Store, "store.verify", "", "nil manifest")
	}
	if m.SchemaVersion != treeManifestSchemaVersion {
		return apperr.New(apperr.Store, "store.verify", "",
			fmt.Sprintf("unsupported manifest schema version %d", m.SchemaVersion))
	}
	seen := make(map[string]struct{}, len(m.Entries))
	collisionSeen := make(map[string]string, len(m.Entries))
	for _, e := range m.Entries {
		if err := validateManifestPath(e.Path); err != nil {
			return apperr.Wrap(apperr.Store, "store.verify", e.Path, err)
		}
		if _, ok := seen[e.Path]; ok {
			return apperr.New(apperr.Store, "store.verify", e.Path, "duplicate manifest path")
		}
		seen[e.Path] = struct{}{}
		if ck := portableCollisionKey(e.Path); ck != "" {
			if first, ok := collisionSeen[ck]; ok {
				return portablePathCollisionErr(first, e.Path)
			}
			collisionSeen[ck] = e.Path
		}
		switch TreeEntryKind(e.Kind) {
		case EntryFile:
			if err := validateManifestHash(e.Hash); err != nil {
				return apperr.Wrap(apperr.Store, "store.verify", e.Path, err)
			}
		case EntrySymlink:
			if err := validateSymlinkTarget(e.SymlinkTarget); err != nil {
				return apperr.Wrap(apperr.Store, "store.verify", e.Path, err)
			}
		case EntryDir:
		default:
			return apperr.New(apperr.Store, "store.verify", e.Path, fmt.Sprintf("unsupported entry kind %q", e.Kind))
		}
	}
	return nil
}

func validateManifestHash(hash string) error {
	if len(hash) != sha256.Size*2 {
		return fmt.Errorf("invalid file hash length")
	}
	for _, c := range hash {
		if !unicode.IsDigit(c) && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return fmt.Errorf("invalid file hash")
		}
	}
	return nil
}

func validateSymlinkTarget(target string) error {
	if target == "" {
		return fmt.Errorf("symlink entry missing target")
	}
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "\\") {
		return fmt.Errorf("absolute symlink target")
	}
	if filepath.IsAbs(target) {
		return fmt.Errorf("absolute symlink target")
	}
	clean := filepath.ToSlash(filepath.Clean(target))
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("escaping symlink target")
	}
	if clean != filepath.ToSlash(target) {
		return fmt.Errorf("non-canonical symlink target")
	}
	for _, seg := range strings.Split(clean, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if isWindowsReservedName(seg) {
			return fmt.Errorf("windows reserved symlink target")
		}
	}
	return nil
}

func validateManifestPath(path string) error {
	if path == "" {
		return fmt.Errorf("empty manifest path")
	}
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return fmt.Errorf("absolute manifest path")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute manifest path")
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("escaping manifest path")
	}
	if clean != filepath.ToSlash(path) {
		return fmt.Errorf("non-canonical manifest path")
	}
	for _, seg := range strings.Split(clean, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if isWindowsReservedName(seg) {
			return fmt.Errorf("windows reserved manifest path")
		}
	}
	return nil
}

func isWindowsReservedName(base string) bool {
	if base == "" {
		return false
	}
	name := strings.ToUpper(strings.TrimSuffix(base, filepath.Ext(base)))
	switch name {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(name) == 4 {
		if strings.HasPrefix(name, "COM") && name[3] >= '1' && name[3] <= '9' {
			return true
		}
		if strings.HasPrefix(name, "LPT") && name[3] >= '1' && name[3] <= '9' {
			return true
		}
	}
	return false
}

func verifyTreeManifest(dir string, m *TreeManifest) error {
	if err := validateTreeManifest(m); err != nil {
		return err
	}
	manifestPaths := make(map[string]TreeEntry, len(m.Entries))
	for _, e := range m.Entries {
		manifestPaths[e.Path] = e
		if err := verifyManifestEntry(dir, e); err != nil {
			return err
		}
	}
	if err := verifyNoExtraTreePaths(dir, manifestPaths); err != nil {
		return err
	}
	if _, err := os.Stat(treeManifestPath(dir)); err != nil {
		return apperr.Wrap(apperr.Store, "store.verify", treeManifestFileName, err)
	}
	if _, err := os.Stat(filepath.Join(dir, packageMarker)); err != nil {
		return apperr.Wrap(apperr.Store, "store.verify", packageMarker, err)
	}
	pkgJSON := filepath.Join(dir, "package.json")
	if _, err := os.Stat(pkgJSON); err != nil {
		return apperr.Wrap(apperr.Store, "store.verify", pkgJSON, err)
	}
	return nil
}

func verifyManifestEntry(dir string, e TreeEntry) error {
	path := filepath.Join(dir, filepath.FromSlash(e.Path))
	info, err := os.Lstat(path)
	if err != nil {
		return apperr.Wrap(apperr.Store, "store.verify", e.Path, err)
	}
	mode := uint32(info.Mode().Perm())
	if mode != e.Mode {
		return apperr.New(apperr.Store, "store.verify", e.Path, "mode drift")
	}
	switch TreeEntryKind(e.Kind) {
	case EntrySymlink:
		if info.Mode()&fs.ModeSymlink == 0 {
			return apperr.New(apperr.Store, "store.verify", e.Path, "expected symlink")
		}
		target, err := os.Readlink(path)
		if err != nil {
			return apperr.Wrap(apperr.Store, "store.verify", e.Path, err)
		}
		if filepath.ToSlash(target) != e.SymlinkTarget {
			return apperr.New(apperr.Store, "store.verify", e.Path, "symlink target changed")
		}
	case EntryFile:
		if info.Mode()&fs.ModeType != 0 && info.Mode()&fs.ModeSymlink == 0 {
			return apperr.New(apperr.Store, "store.verify", e.Path, "expected regular file")
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return apperr.New(apperr.Store, "store.verify", e.Path, "expected file, found symlink")
		}
		got, err := hashFile(path)
		if err != nil {
			return apperr.Wrap(apperr.Store, "store.verify", e.Path, err)
		}
		if !strings.EqualFold(got, e.Hash) {
			return apperr.New(apperr.Store, "store.verify", e.Path, "content hash mismatch")
		}
	case EntryDir:
		if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return apperr.New(apperr.Store, "store.verify", e.Path, "expected directory")
		}
	default:
		return apperr.New(apperr.Store, "store.verify", e.Path, "unknown entry kind")
	}
	return nil
}

func verifyNoExtraTreePaths(dir string, manifestPaths map[string]TreeEntry) error {
	collisionIndex := make(map[string]string, len(manifestPaths))
	for path := range manifestPaths {
		ck := portableCollisionKey(path)
		if first, ok := collisionIndex[ck]; ok && first != path {
			return portablePathCollisionErr(first, path)
		}
		collisionIndex[ck] = path
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		base := filepath.Base(rel)
		if base == treeManifestFileName || base == packageMarker {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, ok := manifestPaths[rel]; ok {
			return nil
		}
		if canonical, ok := collisionIndex[portableCollisionKey(rel)]; ok {
			return portablePathCollisionErr(canonical, rel)
		}
		if d.IsDir() {
			return apperr.New(apperr.Store, "store.verify", rel, "extra directory")
		}
		return apperr.New(apperr.Store, "store.verify", rel, "extra file")
	})
}
