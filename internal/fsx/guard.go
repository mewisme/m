package fsx

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
)

// IsSymlinkOrJunction reports whether fi is a symlink or directory junction.
func IsSymlinkOrJunction(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink != 0
}

// RequiresAncestorGuard reports whether rel touches sensitive project paths.
func RequiresAncestorGuard(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, ".mew/") || rel == ".mew" {
		return true
	}
	if rel == "node_modules" || strings.HasPrefix(rel, "node_modules/") || strings.Contains(rel, "/node_modules/") {
		return true
	}
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		if p == "snapshots" {
			for j := 0; j < i; j++ {
				if parts[j] == ".mew" {
					return true
				}
			}
		}
	}
	return false
}

// GuardAncestors Lstats each existing component from base through target and rejects symlinks/junctions.
func GuardAncestors(base, target string) error {
	base, err := filepath.Abs(base)
	if err != nil {
		return apperr.Wrap(apperr.Transaction, "fs.guard", base, err)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return apperr.Wrap(apperr.Transaction, "fs.guard", target, err)
	}
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	if target != base && !strings.HasPrefix(target, base+string(filepath.Separator)) {
		return apperr.New(apperr.Transaction, "fs.guard", target, "path escapes base")
	}
	if err := lstatNoSymlink(base); err != nil {
		return err
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return apperr.Wrap(apperr.Transaction, "fs.guard", target, err)
	}
	if rel == "." {
		return nil
	}
	cur := base
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return apperr.Wrap(apperr.IO, "fs.guard", cur, err)
		}
		if IsSymlinkOrJunction(fi) && cur != target {
			return apperr.New(apperr.Transaction, "fs.guard", cur, "symlink or junction in guarded path")
		}
	}
	return nil
}

func lstatNoSymlink(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "fs.guard", path, err)
	}
	if IsSymlinkOrJunction(fi) {
		return apperr.New(apperr.Transaction, "fs.guard", path, "symlink or junction in guarded path")
	}
	return nil
}
