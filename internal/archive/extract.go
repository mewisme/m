package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

// ponytail: bomb limits are fixed constants; upgrade = config keys for body/members/expansion.

const (
	maxMembers        = 100_000
	maxExpansionRatio = 10
)

// Options configures safe extraction.
type Options struct {
	StripPackagePrefix bool
}

// DefaultOptions returns npm-oriented defaults.
func DefaultOptions() Options {
	return Options{StripPackagePrefix: true}
}

// Extract unpacks a .tgz into destDir with path traversal guards.
func Extract(ctx context.Context, tgzPath, destDir string, opts Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "archive.extract", destDir, err)
	}
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return apperr.Wrap(apperr.IO, "archive.extract", destDir, err)
	}

	f, err := os.Open(tgzPath)
	if err != nil {
		return apperr.Wrap(apperr.IO, "archive.extract", tgzPath, err)
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return apperr.Wrap(apperr.IO, "archive.extract", tgzPath, err)
	}
	compressedSize := st.Size()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return apperr.Wrap(apperr.Integrity, "archive.extract", tgzPath, err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	var (
		members      int
		uncompressed int64
		epoch        = time.Unix(0, 0).UTC()
	)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return apperr.Wrap(apperr.Integrity, "archive.extract", tgzPath, err)
		}
		members++
		if members > maxMembers {
			return apperr.New(apperr.Integrity, "archive.extract", tgzPath,
				fmt.Sprintf("member count exceeds %d", maxMembers))
		}
		uncompressed += hdr.Size
		if compressedSize > 0 && uncompressed > compressedSize*maxExpansionRatio {
			return apperr.New(apperr.Integrity, "archive.extract", tgzPath,
				fmt.Sprintf("expansion ratio exceeds %dx", maxExpansionRatio))
		}

		name := hdr.Name
		if opts.StripPackagePrefix {
			name = strings.TrimPrefix(name, "package/")
		}
		if name == "" || name == "." {
			continue
		}
		target, err := safeJoin(destAbs, name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir, tar.TypeReg:
			mode := fileMode(hdr)
			if hdr.Typeflag == tar.TypeDir {
				mode = dirMode(hdr)
				if err := os.MkdirAll(target, mode); err != nil {
					return apperr.Wrap(apperr.IO, "archive.extract", target, err)
				}
				_ = os.Chtimes(target, epoch, epoch)
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return apperr.Wrap(apperr.IO, "archive.extract", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return apperr.Wrap(apperr.IO, "archive.extract", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return apperr.Wrap(apperr.IO, "archive.extract", target, err)
			}
			if err := out.Close(); err != nil {
				return apperr.Wrap(apperr.IO, "archive.extract", target, err)
			}
			_ = os.Chtimes(target, epoch, epoch)
		case tar.TypeSymlink, tar.TypeLink:
			link := hdr.Linkname
			if opts.StripPackagePrefix {
				link = strings.TrimPrefix(link, "package/")
			}
			if err := validateLinkTarget(destAbs, target, link, hdr.Typeflag == tar.TypeSymlink); err != nil {
				return err
			}
			if hdr.Typeflag == tar.TypeSymlink {
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return apperr.Wrap(apperr.IO, "archive.extract", target, err)
				}
				if err := os.Symlink(link, target); err != nil {
					return apperr.Wrap(apperr.IO, "archive.extract", target, err)
				}
			} else {
				linkAbs, err := safeJoin(destAbs, link)
				if err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return apperr.Wrap(apperr.IO, "archive.extract", target, err)
				}
				if err := os.Link(linkAbs, target); err != nil {
					return apperr.Wrap(apperr.IO, "archive.extract", target, err)
				}
			}
		default:
			return apperr.New(apperr.Integrity, "archive.extract", hdr.Name,
				fmt.Sprintf("unsupported tar type %c", hdr.Typeflag))
		}
	}
	return nil
}

func safeJoin(destAbs, name string) (string, error) {
	clean := path.Clean(filepath.ToSlash(name))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", apperr.New(apperr.Integrity, "archive.path", name, "path escapes destination")
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return "", apperr.New(apperr.Integrity, "archive.path", name, "absolute path")
	}
	if runtime.GOOS == "windows" {
		if len(name) >= 2 && name[1] == ':' {
			return "", apperr.New(apperr.Integrity, "archive.path", name, "drive path")
		}
		if strings.Contains(strings.ToLower(name), `\windows\`) {
			return "", apperr.New(apperr.Integrity, "archive.path", name, "windows system path")
		}
	}
	target := filepath.Join(destAbs, filepath.FromSlash(clean))
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "archive.path", name, err)
	}
	rel, err := filepath.Rel(destAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", apperr.New(apperr.Integrity, "archive.path", name, "path escapes destination")
	}
	return targetAbs, nil
}

func validateLinkTarget(destAbs, target, link string, symlink bool) error {
	if filepath.IsAbs(link) || strings.HasPrefix(link, "/") {
		return apperr.New(apperr.Integrity, "archive.link", link, "absolute link target")
	}
	if runtime.GOOS == "windows" && len(link) >= 2 && link[1] == ':' {
		return apperr.New(apperr.Integrity, "archive.link", link, "drive link target")
	}
	if symlink {
		base := filepath.Dir(target)
		resolved := filepath.Join(base, link)
		resolvedAbs, err := filepath.Abs(resolved)
		if err != nil {
			return apperr.Wrap(apperr.IO, "archive.link", link, err)
		}
		rel, err := filepath.Rel(destAbs, resolvedAbs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return apperr.New(apperr.Integrity, "archive.link", link, "link escapes destination")
		}
		return nil
	}
	_, err := safeJoin(destAbs, link)
	return err
}

func dirMode(hdr *tar.Header) os.FileMode {
	return 0o755
}

func fileMode(hdr *tar.Header) os.FileMode {
	mode := os.FileMode(0o644)
	if hdr.Mode&0o100 != 0 {
		mode |= 0o111
	}
	return mode
}
