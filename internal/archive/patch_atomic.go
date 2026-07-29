package archive

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/transaction"
)

// ApplyPatchOptions configures atomic patch application in a staging layout.
type ApplyPatchOptions struct {
	SourceRoot  string
	WorkRoot    string
	PublishRoot string
	PatchPath   string
}

// ApplyPatchAtomic copies source to work, applies patch, and renames work to publish.
// On failure the work directory is removed; source and publish roots are untouched.
func ApplyPatchAtomic(ctx context.Context, opts ApplyPatchOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := transaction.InvokeTestHook("post_patch_copy", 0); err != nil {
		return err
	}
	if err := removeDirAll(opts.WorkRoot); err != nil {
		return apperr.Wrap(apperr.IO, "archive.patch.atomic", opts.WorkRoot, err)
	}
	if err := copyDirTree(opts.SourceRoot, opts.WorkRoot); err != nil {
		_ = removeDirAll(opts.WorkRoot)
		return apperr.Wrap(apperr.IO, "archive.patch.atomic", opts.SourceRoot, err)
	}

	if err := transaction.InvokeTestHook("post_patch_preflight", 0); err != nil {
		_ = removeDirAll(opts.WorkRoot)
		return err
	}
	plan, err := PreflightPlan(ctx, opts.PatchPath, opts.WorkRoot)
	if err != nil {
		_ = removeDirAll(opts.WorkRoot)
		return err
	}

	if err := transaction.InvokeTestHook("post_patch_apply", 0); err != nil {
		_ = removeDirAll(opts.WorkRoot)
		return err
	}
	if err := ApplyPlan(ctx, plan); err != nil {
		_ = removeDirAll(opts.WorkRoot)
		return err
	}

	if err := transaction.InvokeTestHook("post_patch_publish", 0); err != nil {
		_ = removeDirAll(opts.WorkRoot)
		return err
	}
	if err := publishPatchWork(opts.WorkRoot, opts.PublishRoot); err != nil {
		_ = removeDirAll(opts.WorkRoot)
		return err
	}
	return nil
}

func publishPatchWork(workRoot, publishRoot string) error {
	if err := removeDirAll(publishRoot); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(publishRoot), 0o755); err != nil {
		return err
	}
	return os.Rename(workRoot, publishRoot)
}

func removeDirAll(path string) error {
	if path == "" {
		return nil
	}
	err := os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CopyDirTree recursively copies a directory tree (regular files only).
func CopyDirTree(src, dst string) error {
	return copyDirTree(src, dst)
}

func copyDirTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "node_modules" {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFilePreserve(path, target)
	})
}

func copyFilePreserve(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
