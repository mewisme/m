package linker

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
)

func copyTree(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target, d.Type())
	})
}

func copyFile(src, dest string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	perm := mode.Perm()
	if perm == 0 {
		perm = 0o644
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// Apply executes mkdir and copy ops in plan order.
func Apply(ctx context.Context, plan *Plan) error {
	if plan == nil {
		return apperr.New(apperr.Internal, "linker.apply", "plan", "nil plan")
	}
	for _, op := range plan.Ops {
		if err := ctx.Err(); err != nil {
			return err
		}
		switch op.Kind {
		case OpMkdir:
			if err := os.MkdirAll(op.Dest, 0o755); err != nil {
				return apperr.Wrap(apperr.IO, "linker.apply", op.Dest, err)
			}
		case OpCopy:
			if op.Src == "" || op.Dest == "" {
				return apperr.New(apperr.Internal, "linker.apply", "copy", "missing src or dest")
			}
			if err := os.MkdirAll(op.Dest, 0o755); err != nil {
				return apperr.Wrap(apperr.IO, "linker.apply", op.Dest, err)
			}
			if err := copyTree(op.Src, op.Dest); err != nil {
				return apperr.Wrap(apperr.IO, "linker.apply", op.Src, err)
			}
		default:
			return apperr.New(apperr.Internal, "linker.apply", "op", "unknown op kind")
		}
	}
	if len(plan.Bins) > 0 {
		if err := WriteBins(plan.NodeModules, plan.Bins); err != nil {
			return err
		}
	}
	return nil
}
