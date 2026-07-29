package pack

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
)

// Options configures deterministic package tarball creation.
type Options struct {
	Root            string // package directory (contains package.json)
	PackDestination string // output directory for .tgz (default Root)
}

// Result summarizes a pack run.
type Result struct {
	TarballPath string
	Files       []string // sorted relative paths inside package/
}

// Pack builds name-version.tgz using npm files/.npmignore rules.
func Pack(ctx context.Context, opts Options) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return Result{}, apperr.Wrap(apperr.IO, "pack", opts.Root, err)
	}
	st, err := os.Stat(root)
	if err != nil {
		return Result{}, apperr.Wrap(apperr.IO, "pack", root, err)
	}
	if !st.IsDir() {
		return Result{}, apperr.New(apperr.Usage, "pack", root, "package path must be a directory")
	}

	doc, err := manifest.Load(filepath.Join(root, "package.json"))
	if err != nil {
		return Result{}, err
	}
	if err := doc.Validate(); err != nil {
		return Result{}, err
	}
	if doc.Name == "" || doc.Version == "" {
		return Result{}, apperr.New(apperr.Manifest, "pack", "package.json", "name and version are required")
	}

	files, err := ListFiles(root, doc.Source)
	if err != nil {
		return Result{}, err
	}

	destDir := opts.PackDestination
	if destDir == "" {
		destDir = root
	}
	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return Result{}, apperr.Wrap(apperr.IO, "pack", destDir, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return Result{}, apperr.Wrap(apperr.IO, "pack", destDir, err)
	}

	tarballName := TarballFileName(doc.Name, doc.Version)
	tarballPath := filepath.Join(destDir, tarballName)

	if err := writeTarball(ctx, tarballPath, root, files); err != nil {
		return Result{}, err
	}
	return Result{TarballPath: tarballPath, Files: files}, nil
}

// ListFiles returns sorted relative paths that would be packed (slash-separated).
func ListFiles(root string, pkgJSON []byte) ([]string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "pack.list", root, err)
	}
	whitelist, err := parseFilesField(pkgJSON)
	if err != nil {
		return nil, err
	}
	ignore := loadIgnorePatterns(root)

	var candidates []string
	if len(whitelist) > 0 {
		candidates, err = collectWhitelist(root, whitelist)
	} else {
		candidates, err = collectAll(root)
	}
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var out []string
	add := func(rel string) {
		rel = filepath.ToSlash(rel)
		if rel == "" || rel == "." {
			return
		}
		if ignoredPath(rel, ignore) {
			return
		}
		if _, ok := seen[rel]; ok {
			return
		}
		seen[rel] = struct{}{}
		out = append(out, rel)
	}

	add("package.json")
	for _, rel := range candidates {
		add(rel)
	}
	sort.Strings(out)
	return out, nil
}

// TarballFileName returns npm-style name-version.tgz.
func TarballFileName(name, version string) string {
	safe := strings.TrimPrefix(name, "@")
	safe = strings.ReplaceAll(safe, "/", "-")
	return safe + "-" + version + ".tgz"
}

func parseFilesField(pkgJSON []byte) ([]string, error) {
	if len(pkgJSON) == 0 {
		return nil, nil
	}
	var raw struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(pkgJSON, &raw); err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "pack", "package.json", err)
	}
	return raw.Files, nil
}

func collectAll(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

func collectWhitelist(root string, entries []string) ([]string, error) {
	var files []string
	seen := map[string]struct{}{}
	for _, entry := range entries {
		entry = filepath.FromSlash(strings.TrimSpace(entry))
		if entry == "" || entry == "package.json" {
			continue
		}
		abs := filepath.Join(root, entry)
		st, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, apperr.Wrap(apperr.IO, "pack", entry, err)
		}
		if st.IsDir() {
			err := filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				rel = filepath.ToSlash(rel)
				if _, ok := seen[rel]; ok {
					return nil
				}
				seen[rel] = struct{}{}
				files = append(files, rel)
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}
		rel := filepath.ToSlash(entry)
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		files = append(files, rel)
	}
	return files, nil
}

func writeTarball(ctx context.Context, dest, root string, files []string) error {
	tmp := dest + ".tmp"
	if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "pack", tmp, err)
	}
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return apperr.Wrap(apperr.IO, "pack", tmp, err)
	}
	if err := writeTarballStream(ctx, f, root, files); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return apperr.Wrap(apperr.IO, "pack", tmp, err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		return apperr.Wrap(apperr.IO, "pack", dest, err)
	}
	return nil
}

func writeTarballStream(ctx context.Context, w io.Writer, root string, files []string) error {
	gz := gzip.NewWriter(w)
	defer func() { _ = gz.Close() }()
	tw := tar.NewWriter(gz)
	defer func() { _ = tw.Close() }()

	epoch := time.Unix(0, 0).UTC()
	dirs := map[string]struct{}{}
	for _, rel := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		parts := strings.Split(rel, "/")
		for i := 1; i < len(parts); i++ {
			dirs[strings.Join(parts[:i], "/")] = struct{}{}
		}
	}
	dirList := make([]string, 0, len(dirs))
	for d := range dirs {
		dirList = append(dirList, d)
	}
	sort.Strings(dirList)
	for _, d := range dirList {
		if err := writeTarDir(tw, "package/"+d, epoch); err != nil {
			return err
		}
	}
	for _, rel := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		src := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(src)
		if err != nil {
			return apperr.Wrap(apperr.IO, "pack", rel, err)
		}
		hdr := &tar.Header{
			Name:     "package/" + rel,
			Mode:     0o644,
			Size:     int64(len(data)),
			ModTime:  epoch,
			Typeflag: tar.TypeReg,
			Format:   tar.FormatUSTAR,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return apperr.Wrap(apperr.IO, "pack", rel, err)
		}
		if _, err := tw.Write(data); err != nil {
			return apperr.Wrap(apperr.IO, "pack", rel, err)
		}
	}
	return nil
}

func writeTarDir(tw *tar.Writer, name string, mod time.Time) error {
	if !strings.HasSuffix(name, "/") {
		name += "/"
	}
	hdr := &tar.Header{
		Name:     name,
		Mode:     0o755,
		ModTime:  mod,
		Typeflag: tar.TypeDir,
		Format:   tar.FormatUSTAR,
	}
	return tw.WriteHeader(hdr)
}

func readFileOptional(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

// ReadPackageJSONFromTarball extracts package/package.json from a .tgz.
func ReadPackageJSONFromTarball(tgzPath string) ([]byte, error) {
	f, err := os.Open(tgzPath)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "pack.read", tgzPath, err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, apperr.Wrap(apperr.Integrity, "pack.read", tgzPath, err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, apperr.Wrap(apperr.Integrity, "pack.read", tgzPath, err)
		}
		name := strings.TrimPrefix(hdr.Name, "package/")
		if name == "package.json" {
			data, err := io.ReadAll(io.LimitReader(tr, 1<<20))
			if err != nil {
				return nil, apperr.Wrap(apperr.IO, "pack.read", "package.json", err)
			}
			return data, nil
		}
	}
	return nil, apperr.New(apperr.Manifest, "pack.read", tgzPath, "package.json missing from tarball")
}

// TarballBytes reads a .tgz file.
func TarballBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "pack.read", path, err)
	}
	if len(data) == 0 {
		return nil, apperr.New(apperr.Integrity, "pack.read", path, "empty tarball")
	}
	return data, nil
}

// NormalizePackageJSON returns compact JSON for publish metadata.
func NormalizePackageJSON(raw []byte) (json.RawMessage, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "pack", "package.json", err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "pack", "package.json", err)
	}
	return out, nil
}

// ValidateTarballName checks tarball filename matches name@version.
func ValidateTarballName(tgzPath, name, version string) error {
	base := filepath.Base(tgzPath)
	want := TarballFileName(name, version)
	if base != want {
		return apperr.New(apperr.Usage, "publish", base,
			fmt.Sprintf("tarball name %q does not match package %s@%s (want %q)", base, name, version, want))
	}
	return nil
}
