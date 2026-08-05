package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/runtime/assets"
)

func TestVerifyCacheAllClean(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	_, err := assets.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}

	// Pre-extract all assets.
	paths, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no assets extracted")
	}

	// Verify should pass.
	if err := VerifyCache(eff); err != nil {
		t.Fatalf("VerifyCache on clean cache: %v", err)
	}
}

func TestVerifyCacheMissingFile(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	_, err := assets.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}

	// Extract assets but then delete one.
	paths, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}
	// Find an extracted asset and delete it.
	var deletedName string
	for name, p := range paths {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		deletedName = name
		break
	}

	// Verify should succeed (missing file is not fatal - will be re-extracted).
	if err := VerifyCache(eff); err != nil {
		t.Fatalf("VerifyCache after delete: %v", err)
	}

	// EnsureAssets should re-extract the deleted file.
	paths2, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := paths2[deletedName]; !ok {
		t.Fatalf("deleted asset %s was not re-extracted", deletedName)
	}
	if _, err := os.Stat(paths2[deletedName]); err != nil {
		t.Fatalf("re-extracted asset not on disk: %v", err)
	}
}

func TestVerifyCacheCorruptFile(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}
	_, err := assets.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}

	paths, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt one extracted file.
	var corruptedName string
	for name, p := range paths {
		if err := os.WriteFile(p, []byte("corrupted content"), 0o644); err != nil {
			t.Fatal(err)
		}
		corruptedName = name
		break
	}

	// Verify should succeed (corrupt file deleted, re-extraction will fix it).
	if err := VerifyCache(eff); err != nil {
		t.Fatalf("VerifyCache on corrupt file: %v", err)
	}

	// Corrupted file should be deleted.
	// EnsureAssets should re-extract the corrupt file.
	paths2, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := paths2[corruptedName]
	if !ok {
		t.Fatalf("corrupted asset %s was not re-extracted", corruptedName)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "corrupted content" {
		t.Fatal("corrupted content was not replaced")
	}
}

func TestVerifyCacheUnreadableAsset(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}

	// Pre-extract assets.
	paths, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}

	// Make one extracted asset unreadable.
	for _, p := range paths {
		if err := os.Chmod(p, 0o200); err != nil {
			t.Fatal(err)
		}
		break
	}

	// Verify should fail: can't read asset file.
	err = VerifyCache(eff)
	if err == nil {
		t.Fatal("expected error for unreadable asset")
	}
}

func TestVerifyCacheSymlinkRejection(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}

	_, err := assets.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}

	// Pre-extract assets.
	paths, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}

	// Replace one asset with a symlink.
	var symlinkName string
	for name, p := range paths {
		// Create a target file.
		targetPath := p + ".real"
		if err := os.WriteFile(targetPath, []byte("real content"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(targetPath, p); err != nil {
			t.Fatal(err)
		}
		symlinkName = name
		break
	}

	// Verify should succeed (symlink deleted, re-extraction will fix it).
	if err := VerifyCache(eff); err != nil {
		t.Fatalf("VerifyCache on symlink: %v", err)
	}

	// Symlink should be gone.
	paths2, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := paths2[symlinkName]
	if !ok {
		t.Fatalf("symlinked asset %s was not re-extracted", symlinkName)
	}
	if isSymlink(p) {
		t.Fatal("re-extracted asset is still a symlink")
	}
}

func TestVerifyCachePartiallyValid(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}

	_, err := assets.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}

	// Extract assets, then corrupt only some of them.
	paths, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt every other asset.
	idx := 0
	for name, p := range paths {
		if idx%2 == 0 {
			if err := os.WriteFile(p, []byte("corrupted"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		idx++
		_ = name
	}

	// Verify should succeed — corrupt files deleted, clean files untouched.
	if err := VerifyCache(eff); err != nil {
		t.Fatalf("VerifyCache on partially corrupt cache: %v", err)
	}

	// Re-extraction should restore corrupted files.
	paths2, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths2) == 0 {
		t.Fatal("no assets after re-extraction")
	}
}

func TestVerifyCacheConcurrentPopulation(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}

	// Run EnsureAssets which uses extractionMu.
	paths1, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}

	// Concurrent VerifyCache should work alongside EnsureAssets.
	done := make(chan error, 1)
	go func() {
		done <- VerifyCache(eff)
	}()

	paths2, err := EnsureAssets(eff)
	if err != nil {
		t.Fatal(err)
	}

	if err := <-done; err != nil {
		t.Fatalf("concurrent VerifyCache: %v", err)
	}

	if len(paths1) == 0 || len(paths2) == 0 {
		t.Fatal("concurrent population failed")
	}
}

func TestCacheEnsureDigestMismatchRejected(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}

	manifest, err := assets.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}

	entries := manifest.AssetsSorted()
	if len(entries) == 0 {
		t.Fatal("no assets in manifest")
	}

	// Create an entry with a wrong digest.
	badEntry := entries[0]
	badEntry.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	cacheDir, err := CacheDir(eff)
	if err != nil {
		t.Fatal(err)
	}

	c := &extractCache{}
	_, err = c.cacheEnsure(cacheDir, badEntry)
	if err == nil {
		t.Fatal("expected digest mismatch error")
	}
}

func TestCleanCacheRemovesStaleVersions(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}

	manifest, err := assets.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}

	// Create a stale version directory.
	runtimeDir := filepath.Join(dir, "runtime")
	staleDir := filepath.Join(runtimeDir, "stale-version")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleDir, "junk"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create the current version directory.
	currentDir := filepath.Join(runtimeDir, manifest.BundleVersion)
	if err := os.MkdirAll(currentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := CleanCache(eff); err != nil {
		t.Fatalf("CleanCache: %v", err)
	}

	// Stale dir should be gone.
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Fatal("stale version directory not removed")
	}
	// Current dir should remain.
	if _, err := os.Stat(currentDir); err != nil {
		t.Fatal("current version directory was removed")
	}
}

func TestCacheEnsureRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	eff := &config.Effective{Values: map[string]config.Value{"cache.dir": {Raw: dir}}}

	manifest, err := assets.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}

	entries := manifest.AssetsSorted()
	if len(entries) == 0 {
		t.Fatal("no assets in manifest")
	}
	entry := entries[0]

	cacheDir, err := CacheDir(eff)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(cacheDir, entry.Path)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}

	// Place a symlink at the destination.
	targetPath := dest + ".real"
	if err := os.WriteFile(targetPath, []byte("real"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetPath, dest); err != nil {
		t.Fatal(err)
	}

	c := &extractCache{}
	resultPath, err := c.cacheEnsure(cacheDir, entry)
	if err != nil {
		t.Fatalf("cacheEnsure should re-extract over symlink: %v", err)
	}
	if isSymlink(resultPath) {
		t.Fatal("result is still a symlink after cacheEnsure")
	}

	// Verify the extracted content is correct.
	data, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != entry.SHA256 {
		t.Fatal("extracted content digest mismatch")
	}
}

func TestEnsureAssetsNilEffectiveConfig(t *testing.T) {
	_, err := EnsureAssets(nil)
	if err == nil {
		t.Fatal("expected error for nil effective config")
	}
}

func TestCacheDirNilEffectiveConfig(t *testing.T) {
	_, err := CacheDir(nil)
	if err == nil {
		t.Fatal("expected error for nil effective config")
	}
}
