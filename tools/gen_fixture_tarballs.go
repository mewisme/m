//go:build ignore

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type blob struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Blobs         []blob `json:"blobs"`
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	reg := filepath.Join(root, "fixtures", "registry", "v1")
	tarDir := filepath.Join(reg, "tarballs")
	_ = os.MkdirAll(tarDir, 0o755)

	specs := []struct {
		file, name, version string
		extra               map[string]any
	}{
		{"pkg-a-1.0.0.tgz", "pkg-a", "1.0.0", map[string]any{"dependencies": map[string]string{"pkg-b": "^1.0.0"}}},
		{"pkg-b-1.2.0.tgz", "pkg-b", "1.2.0", map[string]any{"dependencies": map[string]string{"pkg-c": "^1.0.0"}}},
		{"pkg-c-1.0.1.tgz", "pkg-c", "1.0.1", nil},
		{"pkg-1.0.0.tgz", "@scope/pkg", "1.0.0", nil},
	}
	checksums := map[string]string{}
	for _, s := range specs {
		sum, err := writeTarball(filepath.Join(tarDir, s.file), s.name, s.version, s.extra)
		if err != nil {
			panic(err)
		}
		checksums[s.file] = sum
		fmt.Printf("wrote %s sha256=%s\n", s.file, sum)
	}
	updatePackumentIntegrity(reg, "pkg-a", "1.0.0", checksums["pkg-a-1.0.0.tgz"])
	updatePackumentIntegrity(reg, "pkg-b", "1.2.0", checksums["pkg-b-1.2.0.tgz"])
	updatePackumentIntegrity(reg, "pkg-c", "1.0.1", checksums["pkg-c-1.0.1.tgz"])
	updatePackumentIntegrity(reg, "@scope/pkg", "1.0.0", checksums["pkg-1.0.0.tgz"])
	updateManifest(reg, checksums)
	rehashAllBlobs(reg)
}

func rehashAllBlobs(reg string) {
	path := filepath.Join(reg, "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var man manifest
	if err := json.Unmarshal(raw, &man); err != nil {
		panic(err)
	}
	for i, b := range man.Blobs {
		data, err := os.ReadFile(filepath.Join(reg, filepath.FromSlash(b.Path)))
		if err != nil {
			panic(err)
		}
		sum := sha256.Sum256(data)
		man.Blobs[i].SHA256 = hex.EncodeToString(sum[:])
	}
	out, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		panic(err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		panic(err)
	}
}

func writeTarball(path, name, version string, extra map[string]any) (string, error) {
	meta := map[string]any{"name": name, "version": version}
	for k, v := range extra {
		meta[k] = v
	}
	body, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "package/package.json", Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		return "", err
	}
	if _, err := tw.Write(body); err != nil {
		return "", err
	}
	js := fmt.Sprintf("module.exports = %q;\n", name)
	hdr = &tar.Header{Name: "package/index.js", Mode: 0o644, Size: int64(len(js)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		return "", err
	}
	if _, err := tw.Write([]byte(js)); err != nil {
		return "", err
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}
	data := buf.Bytes()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func updatePackumentIntegrity(reg, name, version, sha string) {
	var path string
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name[1:], "/", 2)
		path = filepath.Join(reg, "packuments", "@"+parts[0], parts[1]+".json")
	} else {
		path = filepath.Join(reg, "packuments", name+".json")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		panic(err)
	}
	versions := doc["versions"].(map[string]any)
	ver := versions[version].(map[string]any)
	dist := ver["dist"].(map[string]any)
	dist["integrity"] = "sha256-" + sha
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		panic(err)
	}
}

func updateManifest(reg string, tarballSums map[string]string) {
	path := filepath.Join(reg, "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var man manifest
	if err := json.Unmarshal(raw, &man); err != nil {
		panic(err)
	}
	existing := map[string]blob{}
	for _, b := range man.Blobs {
		existing[b.Path] = b
	}
	for file, sum := range tarballSums {
		key := "tarballs/" + file
		existing[key] = blob{Path: key, SHA256: sum}
	}
	man.Blobs = make([]blob, 0, len(existing))
	for _, b := range existing {
		man.Blobs = append(man.Blobs, b)
	}
	// stable sort
	for i := 0; i < len(man.Blobs); i++ {
		for j := i + 1; j < len(man.Blobs); j++ {
			if man.Blobs[j].Path < man.Blobs[i].Path {
				man.Blobs[i], man.Blobs[j] = man.Blobs[j], man.Blobs[i]
			}
		}
	}
	out, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		panic(err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		panic(err)
	}
}
