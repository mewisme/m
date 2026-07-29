package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// ponytail: fixed pack limits; upgrade = config keys for body/members/expansion.

const (
	maxPackFiles      = 100_000
	maxPackFileBytes  = 512 << 20 // 512 MiB
	maxPackTotalBytes = 2 << 30   // 2 GiB
	maxPackPathLen    = 4096
)

var hardExcludeDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	".mew":         {},
}

// packSandbox enforces root containment while packing.
type packSandbox struct {
	rootAbs    string
	excludeAbs map[string]struct{}
	fileCount  int
	totalBytes int64
}

func newPackSandbox(root string, excludeAbs ...string) (*packSandbox, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "pack.sandbox", root, err)
	}
	exclude := make(map[string]struct{}, len(excludeAbs))
	for _, p := range excludeAbs {
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "pack.sandbox", p, err)
		}
		exclude[filepath.Clean(abs)] = struct{}{}
	}
	return &packSandbox{rootAbs: filepath.Clean(rootAbs), excludeAbs: exclude}, nil
}

func validatePackRel(rel string) error {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return apperr.New(apperr.Integrity, "pack.sandbox", rel, "empty relative path")
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return apperr.New(apperr.Integrity, "pack.sandbox", rel, "empty relative path")
	}
	if len(rel) > maxPackPathLen {
		return apperr.New(apperr.Integrity, "pack.sandbox", rel,
			fmt.Sprintf("path exceeds %d bytes", maxPackPathLen))
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		return apperr.New(apperr.Integrity, "pack.sandbox", rel, "absolute path")
	}
	if runtime.GOOS == "windows" {
		if len(rel) >= 2 && rel[1] == ':' {
			return apperr.New(apperr.Integrity, "pack.sandbox", rel, "drive path")
		}
		if strings.HasPrefix(rel, "//") || strings.HasPrefix(rel, "\\\\") {
			return apperr.New(apperr.Integrity, "pack.sandbox", rel, "unc path")
		}
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return apperr.New(apperr.Integrity, "pack.sandbox", rel, "path escapes package root")
		}
	}
	return nil
}

func (s *packSandbox) resolve(rel string) (string, error) {
	if err := validatePackRel(rel); err != nil {
		return "", err
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	abs := filepath.Join(s.rootAbs, filepath.FromSlash(rel))
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "pack.sandbox", rel, err)
	}
	abs = filepath.Clean(abs)
	relCheck, err := filepath.Rel(s.rootAbs, abs)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		return "", apperr.New(apperr.Integrity, "pack.sandbox", rel, "path escapes package root")
	}
	if err := fsx.GuardAncestors(s.rootAbs, abs); err != nil {
		return "", apperr.Wrap(apperr.Integrity, "pack.sandbox", rel, err)
	}
	return abs, nil
}

func (s *packSandbox) excluded(abs string) bool {
	_, ok := s.excludeAbs[filepath.Clean(abs)]
	return ok
}

func isHardExcludedDir(name string) bool {
	_, ok := hardExcludeDirs[name]
	return ok
}

func isHardExcludedRel(rel string) bool {
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	if len(parts) > 0 {
		if _, ok := hardExcludeDirs[parts[0]]; ok {
			return true
		}
	}
	for _, p := range parts {
		if _, ok := hardExcludeDirs[p]; ok {
			return true
		}
	}
	base := parts[len(parts)-1]
	return strings.HasSuffix(base, ".tmp")
}

func (s *packSandbox) lstatRegular(abs, subject string) (os.FileInfo, error) {
	fi, err := os.Lstat(abs)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "pack.sandbox", subject, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 || fsx.IsSymlinkOrJunction(fi) || fsx.ReparseTag(abs) != 0 {
		return nil, apperr.New(apperr.Integrity, "pack.sandbox", subject, "symlink or reparse point")
	}
	if !fi.Mode().IsRegular() {
		return nil, apperr.New(apperr.Integrity, "pack.sandbox", subject, "not a regular file")
	}
	return fi, nil
}

func (s *packSandbox) accountFile(size int64, subject string) error {
	s.fileCount++
	if s.fileCount > maxPackFiles {
		return apperr.New(apperr.Integrity, "pack.sandbox", subject,
			fmt.Sprintf("file count exceeds %d", maxPackFiles))
	}
	if size > maxPackFileBytes {
		return apperr.New(apperr.Integrity, "pack.sandbox", subject,
			fmt.Sprintf("file size exceeds %d bytes", maxPackFileBytes))
	}
	s.totalBytes += size
	if s.totalBytes > maxPackTotalBytes {
		return apperr.New(apperr.Integrity, "pack.sandbox", subject,
			fmt.Sprintf("total packed bytes exceed %d", maxPackTotalBytes))
	}
	return nil
}

func (s *packSandbox) readFile(rel string) ([]byte, os.FileMode, error) {
	abs, err := s.resolve(rel)
	if err != nil {
		return nil, 0, err
	}
	if s.excluded(abs) {
		return nil, 0, apperr.New(apperr.Integrity, "pack.sandbox", rel, "excluded path")
	}
	fi, err := s.lstatRegular(abs, rel)
	if err != nil {
		return nil, 0, err
	}
	if err := s.accountFile(fi.Size(), rel); err != nil {
		return nil, 0, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, 0, apperr.Wrap(apperr.IO, "pack.sandbox", rel, err)
	}
	if int64(len(data)) != fi.Size() {
		return nil, 0, apperr.New(apperr.Integrity, "pack.sandbox", rel, "file size changed during read")
	}
	return data, tarFileMode(fi), nil
}

func tarFileMode(fi os.FileInfo) os.FileMode {
	perm := fi.Mode().Perm()
	if perm == 0 {
		return 0o644
	}
	mode := perm & 0o777
	if mode > 0o755 {
		mode &= 0o755
	}
	if mode&0o111 != 0 {
		return mode
	}
	return mode &^ 0o111
}

func collectAllSandboxed(s *packSandbox) ([]string, error) {
	var files []string
	err := filepath.WalkDir(s.rootAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == s.rootAbs {
				return nil
			}
			if isHardExcludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(s.rootAbs, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if isHardExcludedRel(rel) {
			return nil
		}
		if err := validatePackRel(rel); err != nil {
			return err
		}
		abs, err := s.resolve(rel)
		if err != nil {
			return err
		}
		if s.excluded(abs) {
			return nil
		}
		if _, err := s.lstatRegular(abs, rel); err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

func collectWhitelistSandboxed(s *packSandbox, entries []string) ([]string, error) {
	var files []string
	seen := map[string]struct{}{}
	for _, entry := range entries {
		entry = filepath.FromSlash(strings.TrimSpace(entry))
		if entry == "" || entry == "package.json" {
			continue
		}
		if err := validatePackRel(entry); err != nil {
			return nil, err
		}
		abs, err := s.resolve(entry)
		if err != nil {
			return nil, err
		}
		if s.excluded(abs) {
			continue
		}
		fi, err := os.Lstat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, apperr.Wrap(apperr.IO, "pack", entry, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 || fsx.IsSymlinkOrJunction(fi) || fsx.ReparseTag(abs) != 0 {
			return nil, apperr.New(apperr.Integrity, "pack.sandbox", entry, "symlink or reparse point")
		}
		if fi.IsDir() {
			err := filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if path != abs && isHardExcludedDir(d.Name()) {
						return filepath.SkipDir
					}
					return nil
				}
				rel, err := filepath.Rel(s.rootAbs, path)
				if err != nil {
					return err
				}
				rel = filepath.ToSlash(rel)
				if isHardExcludedRel(rel) {
					return nil
				}
				if err := validatePackRel(rel); err != nil {
					return err
				}
				fileAbs, err := s.resolve(rel)
				if err != nil {
					return err
				}
				if s.excluded(fileAbs) {
					return nil
				}
				if _, err := s.lstatRegular(fileAbs, rel); err != nil {
					return err
				}
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
		if _, err := s.lstatRegular(abs, entry); err != nil {
			return nil, err
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

func packExcludePaths(root, tarballPath string) []string {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil
	}
	tarAbs, err := filepath.Abs(tarballPath)
	if err != nil {
		return nil
	}
	rel, err := filepath.Rel(rootAbs, tarAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil
	}
	return []string{tarAbs}
}
