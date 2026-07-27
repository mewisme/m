package transaction

import (
	"io"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
)

func backupTree(src, dst string) error {
	visited := make(map[string]struct{})
	return backupTreeEntry(src, dst, visited)
}

func restoreTree(src, dst string) error {
	visited := make(map[string]struct{})
	if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "transaction.restore", dst, err)
	}
	return restoreTreeEntry(src, dst, visited)
}

func backupTreeEntry(src, dst string, visited map[string]struct{}) error {
	info, err := os.Lstat(src)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.backup", src, err)
	}
	if err := rejectUnsupportedEntry(src, info); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return apperr.Wrap(apperr.IO, "transaction.backup", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.backup", dst, err)
		}
		if fsx.IsJunction(src) {
			return createJunction(dst, target)
		}
		return os.Symlink(target, dst)
	}
	if info.IsDir() {
		key, ok := inodeVisitKey(info)
		if ok {
			if _, seen := visited[key]; seen {
				return apperr.New(apperr.Transaction, "transaction.backup", src, "directory cycle detected")
			}
			visited[key] = struct{}{}
			defer delete(visited, key)
		}
		if err := os.MkdirAll(dst, info.Mode()&0o777|0o700); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.backup", dst, err)
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return apperr.Wrap(apperr.IO, "transaction.backup", src, err)
		}
		for _, ent := range entries {
			if err := backupTreeEntry(filepath.Join(src, ent.Name()), filepath.Join(dst, ent.Name()), visited); err != nil {
				return err
			}
		}
		return nil
	}
	return copyRegularFile(src, dst, info.Mode())
}

func restoreTreeEntry(src, dst string, visited map[string]struct{}) error {
	info, err := os.Lstat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "transaction.restore", src, err)
	}
	if err := rejectUnsupportedEntry(src, info); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return apperr.Wrap(apperr.IO, "transaction.restore", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.restore", dst, err)
		}
		if fsx.IsJunction(src) {
			return createJunction(dst, target)
		}
		return os.Symlink(target, dst)
	}
	if info.IsDir() {
		key, ok := inodeVisitKey(info)
		if ok {
			if _, seen := visited[key]; seen {
				return apperr.New(apperr.Transaction, "transaction.restore", src, "directory cycle detected")
			}
			visited[key] = struct{}{}
			defer delete(visited, key)
		}
		if err := os.MkdirAll(dst, info.Mode()&0o777|0o700); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.restore", dst, err)
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return apperr.Wrap(apperr.IO, "transaction.restore", src, err)
		}
		for _, ent := range entries {
			if err := restoreTreeEntry(filepath.Join(src, ent.Name()), filepath.Join(dst, ent.Name()), visited); err != nil {
				return err
			}
		}
		return nil
	}
	return copyRegularFile(src, dst, info.Mode())
}

func copyRegularFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", dst, err)
	}
	in, err := os.Open(src)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", src, err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode&0o777)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", dst, err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", dst, err)
	}
	return nil
}

func rejectUnsupportedEntry(path string, info os.FileInfo) error {
	mode := info.Mode()
	switch {
	case mode&os.ModeDevice != 0:
		return apperr.New(apperr.Transaction, "transaction.backup", path, "device nodes are not supported")
	case mode&os.ModeSocket != 0:
		return apperr.New(apperr.Transaction, "transaction.backup", path, "socket nodes are not supported")
	case mode&os.ModeNamedPipe != 0:
		return apperr.New(apperr.Transaction, "transaction.backup", path, "pipe nodes are not supported")
	default:
		return nil
	}
}
