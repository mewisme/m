package fixturemeta

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileSHA256 returns lowercase hex SHA-256 of a file.
func FileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return FileSHA256Bytes(data), nil
}

// FileSHA256Bytes returns lowercase hex SHA-256 of raw bytes.
func FileSHA256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SourceTreeDigest returns a deterministic digest of all files under root.
// Paths are sorted with forward slashes; metadata.json is excluded when present.
func SourceTreeDigest(root string) (string, error) {
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "metadata.json" {
			return nil
		}
		hash, err := FileSHA256(path)
		if err != nil {
			return err
		}
		entries = append(entries, rel+"\x00"+hash)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	h := sha256.New()
	for _, line := range entries {
		if _, err := io.WriteString(h, line+"\n"); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CollectWorkspaceManifests returns package.json paths under packages/ relative to root.
func CollectWorkspaceManifests(root string) (map[string]string, error) {
	packagesDir := filepath.Join(root, "packages")
	info, err := os.Stat(packagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("packages is not a directory")
	}
	out := make(map[string]string)
	err = filepath.Walk(packagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != "package.json" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		hash, err := FileSHA256(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = hash
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// CollectPatchFiles returns patch file paths relative to root with SHA-256 digests.
func CollectPatchFiles(root string) (map[string]string, error) {
	patchesDir := filepath.Join(root, "patches")
	info, err := os.Stat(patchesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}
	out := make(map[string]string)
	err = filepath.Walk(patchesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".patch") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		hash, err := FileSHA256(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = hash
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
