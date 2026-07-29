package linker

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
)

// Apply executes plan ops in order and writes bin shims.
func Apply(ctx context.Context, plan *Plan) error {
	if plan == nil {
		return apperr.New(apperr.Internal, "linker.apply", "plan", "nil plan")
	}
	for _, op := range plan.Ops {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := applyOp(op); err != nil {
			return err
		}
	}
	if len(plan.Bins) > 0 {
		byNM := map[string][]BinSource{}
		for _, src := range plan.Bins {
			nm := src.NodeModules
			if nm == "" {
				nm = plan.NodeModules
			}
			byNM[nm] = append(byNM[nm], src)
		}
		for nm, sources := range byNM {
			if err := WriteBins(nm, sources); err != nil {
				return err
			}
		}
	}
	plan.LinkSummary.TallyFromOps(plan.Ops)
	return nil
}

func applyOp(op Op) error {
	switch op.Kind {
	case OpMkdir:
		if err := os.MkdirAll(op.Dest, 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "linker.apply", op.Dest, err)
		}
	case OpCopy:
		return applyTree(op.Src, op.Dest, applyCopyFile)
	case OpHardlink:
		return applyTree(op.Src, op.Dest, applyHardlinkFile)
	case OpReflink:
		return applyTree(op.Src, op.Dest, applyReflinkFile)
	case OpSymlink:
		return applySymlink(op.Src, op.Dest)
	case OpJunction:
		return applyJunction(op.Src, op.Dest)
	default:
		return apperr.New(apperr.Internal, "linker.apply", "op", "unknown op kind")
	}
	return nil
}

type fileApplyFn func(src, dest string, mode fs.FileMode) error

func applyTree(src, dest string, fn fileApplyFn) error {
	if src == "" || dest == "" {
		return apperr.New(apperr.Internal, "linker.apply", "tree", "missing src or dest")
	}
	if err := os.RemoveAll(dest); err != nil {
		return apperr.Wrap(apperr.IO, "linker.apply", dest, err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "linker.apply", dest, err)
	}
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
		if d.IsDir() && d.Name() == "node_modules" {
			return filepath.SkipDir
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			_ = os.Remove(target)
			return os.Symlink(link, target)
		}
		if err := fn(path, target, d.Type()); err != nil {
			return applyCopyFile(path, target, d.Type())
		}
		return nil
	})
}

func applyCopyFile(src, dest string, mode fs.FileMode) error {
	return copyFile(src, dest, mode)
}

func applyHardlinkFile(src, dest string, _ fs.FileMode) error {
	_ = os.Remove(dest)
	if err := os.Link(src, dest); err != nil {
		return err
	}
	return nil
}

func applyReflinkFile(src, dest string, _ fs.FileMode) error {
	if err := reflinkFile(src, dest); err != nil {
		return err
	}
	return nil
}

func applySymlink(src, dest string) error {
	if src == "" || dest == "" {
		return apperr.New(apperr.Internal, "linker.apply", "symlink", "missing src or dest")
	}
	_ = os.Remove(dest)
	target := src
	if rel, err := filepath.Rel(filepath.Dir(dest), src); err == nil {
		target = rel
	}
	if err := os.Symlink(target, dest); err != nil {
		return apperr.Wrap(apperr.IO, "linker.apply", dest, err)
	}
	return nil
}

func applyJunction(src, dest string) error {
	if src == "" || dest == "" {
		return apperr.New(apperr.Internal, "linker.apply", "junction", "missing src or dest")
	}
	if err := junctionDir(src, dest); err != nil {
		return applyTree(src, dest, applyCopyFile)
	}
	if _, err := os.Stat(filepath.Join(dest, "package.json")); err != nil {
		_ = os.RemoveAll(dest)
		return applyTree(src, dest, applyCopyFile)
	}
	return nil
}
