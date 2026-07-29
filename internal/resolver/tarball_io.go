package resolver

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/manifest"
)

func (s *resolveState) expandLocalManifestAbs(fromKey, absDir string, depth int, namePath, envKeys []string) error {
	doc, err := manifest.Load(filepath.Join(absDir, "package.json"))
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.local", absDir, err)
	}
	norm, err := manifest.ToNormalized(doc)
	if err != nil {
		return err
	}
	nextNamePath := append(append([]string(nil), namePath...), parsePackageKey(fromKey).Name)
	nextEnv := append(append([]string(nil), envKeys...), fromKey)
	return s.seedDeps(fromKey, "", norm, depth, nextNamePath, nextEnv)
}

func readTarballPackage(tgzPath, fallbackName string) (name, version string, err error) {
	raw, err := readTarballFile(tgzPath, "package.json")
	if err != nil {
		return "", "", err
	}
	var doc struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", "", apperr.Wrap(apperr.Manifest, "resolver.tarball", tgzPath, err)
	}
	name = doc.Name
	if name == "" {
		name = fallbackName
	}
	version = doc.Version
	if version == "" {
		version = "0.0.0"
	}
	return name, version, nil
}

func readTarballFile(tgzPath, want string) ([]byte, error) {
	f, err := os.Open(tgzPath)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "resolver.tarball", tgzPath, err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, apperr.Wrap(apperr.Integrity, "resolver.tarball", tgzPath, err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	want = strings.TrimPrefix(filepath.ToSlash(want), "./")
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "resolver.tarball", tgzPath, err)
		}
		name := strings.TrimPrefix(filepath.ToSlash(hdr.Name), "./")
		if i := strings.IndexByte(name, '/'); i >= 0 {
			name = name[i+1:]
		}
		if name != want {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			return nil, apperr.New(apperr.Integrity, "resolver.tarball", want, "not a regular file")
		}
		const maxJSON = 1 << 20
		b, err := io.ReadAll(io.LimitReader(tr, maxJSON+1))
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "resolver.tarball", want, err)
		}
		if int64(len(b)) > maxJSON {
			return nil, apperr.New(apperr.Integrity, "resolver.tarball", want, "package.json too large")
		}
		return b, nil
	}
	return nil, apperr.New(apperr.NotFound, "resolver.tarball", want, "package.json not in tarball")
}

func extractTarballPeek(ctx context.Context, tgzPath string) (string, error) {
	dir, err := os.MkdirTemp("", "mew-tarball-peek-*")
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "resolver.tarball", tgzPath, err)
	}
	if err := archive.Extract(ctx, tgzPath, dir, archive.DefaultOptions()); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}
