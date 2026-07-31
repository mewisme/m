// Package assets holds embedded Node loader and preload sources.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

//go:embed preload.cjs
var PreloadCJS []byte

//go:embed preload.mjs
var PreloadMJS []byte

//go:embed manifest.json
var manifestRaw []byte

//go:embed preload.cjs preload.mjs manifest.json
var runtimeFS embed.FS

// ManifestEntry is a single asset entry in the runtime manifest.
type ManifestEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	ModuleType string `json:"moduleType"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
}

// AssetManifest lists all embedded runtime assets.
type AssetManifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	BundleVersion string          `json:"bundleVersion"`
	Assets        []ManifestEntry `json:"assets"`
}

// LoadManifest reads and validates the embedded manifest.
func LoadManifest() (*AssetManifest, error) {
	var m AssetManifest
	if err := json.Unmarshal(manifestRaw, &m); err != nil {
		return nil, apperr.Wrap(apperr.RuntimeAssetManifest, "assets.manifest", "", err)
	}
	if m.SchemaVersion != 1 {
		return nil, apperr.New(apperr.RuntimeAssetManifest, "assets.manifest", "",
			fmt.Sprintf("unsupported manifest schema version %d", m.SchemaVersion))
	}
	if m.BundleVersion == "" {
		return nil, apperr.New(apperr.RuntimeAssetManifest, "assets.manifest", "", "missing bundle version")
	}
	return &m, nil
}

// AssetsSorted returns the sorted list of manifest entries.
func (m *AssetManifest) AssetsSorted() []ManifestEntry {
	sorted := make([]ManifestEntry, len(m.Assets))
	copy(sorted, m.Assets)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

// ReadAsset reads an embedded asset by name.
func ReadAsset(name string) ([]byte, error) {
	data, err := fs.ReadFile(runtimeFS, name)
	if err != nil {
		return nil, apperr.Wrap(apperr.RuntimeAssetCache, "assets.read", name, err)
	}
	return data, nil
}

// VerifyAsset checks an extracted asset against the expected SHA-256 digest.
func VerifyAsset(r io.Reader, expectedSHA256 string) error {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return apperr.Wrap(apperr.RuntimeAssetDigest, "assets.verify", "", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expectedSHA256) {
		return apperr.New(apperr.RuntimeAssetDigest, "assets.verify", "",
			fmt.Sprintf("digest mismatch: expected %s, got %s", expectedSHA256, got))
	}
	return nil
}
