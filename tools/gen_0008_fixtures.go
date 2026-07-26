//go:build ignore

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func writeFile(path, content string) {
	must(os.MkdirAll(filepath.Dir(path), 0o755))
	must(os.WriteFile(path, []byte(content), 0o644))
}

func sha256File(path string) string {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	must(err)
	return hex.EncodeToString(h.Sum(nil))
}

func writeTarball(path string, files map[string]string) {
	must(os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	mod := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, name := range names {
		body := []byte(files[name])
		hdr := &tar.Header{
			Name:    name,
			Mode:    0o644,
			Size:    int64(len(body)),
			ModTime: mod,
		}
		must(tw.WriteHeader(hdr))
		_, err := tw.Write(body)
		must(err)
	}
}

func main() {
	root := "."
	reg := filepath.Join(root, "fixtures", "registry", "v1")
	tgz := filepath.Join(reg, "tarballs", "lodash-4.17.21.tgz")
	writeTarball(tgz, map[string]string{
		"package/package.json": "{\n  \"name\": \"lodash\",\n  \"version\": \"4.17.21\",\n  \"main\": \"index.js\"\n}\n",
		"package/index.js":     "module.exports = function lodash() { return 'fixture'; };\n",
	})
	sum := sha256File(tgz)

	packument := map[string]any{
		"name": "lodash",
		"versions": map[string]any{
			"4.17.21": map[string]any{
				"name":    "lodash",
				"version": "4.17.21",
				"dist": map[string]any{
					"tarball":   "lodash-4.17.21.tgz",
					"shasum":    sum[:40],
					"integrity": "sha256-" + sum,
				},
			},
		},
		"dist-tags": map[string]string{"latest": "4.17.21"},
	}
	pb, err := json.MarshalIndent(packument, "", "  ")
	must(err)
	pb = append(pb, '\n')
	pPath := filepath.Join(reg, "packuments", "lodash.json")
	must(os.MkdirAll(filepath.Dir(pPath), 0o755))
	must(os.WriteFile(pPath, pb, 0o644))

	type blob struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	manifest := map[string]any{
		"schemaVersion": 1,
		"blobs": []blob{
			{Path: "packuments/lodash.json", SHA256: sha256File(pPath)},
			{Path: "tarballs/lodash-4.17.21.tgz", SHA256: sum},
		},
	}
	mb, err := json.MarshalIndent(manifest, "", "  ")
	must(err)
	mb = append(mb, '\n')
	must(os.WriteFile(filepath.Join(reg, "manifest.json"), mb, 0o644))
	fmt.Println("registry fixtures ready", sum)

	evilPath := filepath.Join(root, "fixtures", "security", "evil-archives", "path-traversal-members.txt")
	writeFile(evilPath, strings.Join([]string{
		"../../etc/passwd",
		"package/../../../tmp/evil",
		`C:\Windows\System32\evil.dll`,
	}, "\n")+"\n")
	writeFile(filepath.Join(root, "fixtures", "security", "evil-archives", "README.md"),
		"# Evil archives\n\nKnown-bad path members for archive parser fail-closed tests.\n"+
			"Never extract these into production paths. The `path-traversal-members.txt`\n"+
			"lists hostile member names; a real crafted `.tgz` may be added when\n"+
			"`internal/archive` ships extraction.\n")

	writeFile(filepath.Join(root, "fixtures", "projects", "basic-cjs", "package.json"),
		"{\n  \"name\": \"basic-cjs\",\n  \"version\": \"1.0.0\",\n  \"main\": \"index.js\",\n  \"dependencies\": {\n    \"lodash\": \"4.17.21\"\n  }\n}\n")
	writeFile(filepath.Join(root, "fixtures", "projects", "basic-cjs", "index.js"),
		"const _ = require('lodash');\nmodule.exports = _;\n")

	writeFile(filepath.Join(root, "fixtures", "projects", "basic-esm", "package.json"),
		"{\n  \"name\": \"basic-esm\",\n  \"version\": \"1.0.0\",\n  \"type\": \"module\",\n  \"dependencies\": {\n    \"lodash\": \"4.17.21\"\n  }\n}\n")
	writeFile(filepath.Join(root, "fixtures", "projects", "basic-esm", "index.js"),
		"import _ from 'lodash';\nexport default _;\n")

	writeFile(filepath.Join(root, "fixtures", "projects", "typescript-app", "package.json"),
		"{\n  \"name\": \"typescript-app\",\n  \"version\": \"1.0.0\",\n  \"dependencies\": {\n    \"lodash\": \"4.17.21\"\n  }\n}\n")
	writeFile(filepath.Join(root, "fixtures", "projects", "typescript-app", "tsconfig.json"),
		"{\n  \"compilerOptions\": {\n    \"target\": \"ES2020\",\n    \"module\": \"commonjs\",\n    \"strict\": true,\n    \"outDir\": \"dist\"\n  },\n  \"include\": [\"src\"]\n}\n")
	writeFile(filepath.Join(root, "fixtures", "projects", "typescript-app", "src", "index.ts"),
		"export const n: number = 1;\n")

	writeFile(filepath.Join(root, "fixtures", "projects", "workspace-simple", "package.json"),
		"{\n  \"name\": \"workspace-simple\",\n  \"private\": true,\n  \"workspaces\": [\"packages/*\"]\n}\n")
	writeFile(filepath.Join(root, "fixtures", "projects", "workspace-simple", "packages", "a", "package.json"),
		"{\n  \"name\": \"a\",\n  \"version\": \"1.0.0\",\n  \"dependencies\": {\n    \"lodash\": \"4.17.21\"\n  }\n}\n")
	writeFile(filepath.Join(root, "fixtures", "projects", "workspace-simple", "packages", "a", "index.js"),
		"module.exports = 'a';\n")

	writeFile(filepath.Join(root, "fixtures", "registry", "v1", "README.md"),
		"# Fixture registry v1\n\nSynthetic packages for hermetic tests. Never fetched from the public npm registry.\n"+
			"Checksums live in `manifest.json`.\n")
}
