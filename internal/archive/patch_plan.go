package archive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/mewisme/mew/internal/apperr"
)

// Plan is a validated patch application plan.
type Plan struct {
	PatchFile string
	Root      string
	Entries   []PlanEntry
}

// PlanEntry describes one in-place file modification.
type PlanEntry struct {
	Target string
	File   *gitdiff.File
	Mode   os.FileMode
}

// PreflightPlan parses and validates a patch without writing files.
func PreflightPlan(ctx context.Context, patchFile, root string) (Plan, error) {
	data, err := readPatchBytes(ctx, patchFile)
	if err != nil {
		return Plan{}, err
	}
	files, err := parsePatch(ctx, patchFile, data)
	if err != nil {
		return Plan{}, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return Plan{}, apperr.Wrap(apperr.IO, "archive.patch", root, err)
	}
	plan := Plan{PatchFile: patchFile, Root: rootAbs}
	seenTarget := make(map[string]struct{}, len(files))
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return Plan{}, err
		}
		if err := classifyPatchFile(patchFile, f); err != nil {
			return Plan{}, err
		}
		pathName := patchTargetName(f)
		target, err := resolvePatchTarget(patchFile, rootAbs, pathName)
		if err != nil {
			return Plan{}, err
		}
		if _, dup := seenTarget[target]; dup {
			return Plan{}, apperr.New(apperr.Integrity, "archive.patch", patchFile, "conflicting patch targets")
		}
		seenTarget[target] = struct{}{}
		fi, err := os.Lstat(target)
		if err != nil {
			return Plan{}, apperr.Wrap(apperr.IO, "archive.patch", target,
				fmt.Errorf("patch target %q: %w", pathName, err))
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return Plan{}, apperr.New(apperr.Integrity, "archive.patch", pathName, "unsupported patch operation: symlink target")
		}
		if !fi.Mode().IsRegular() {
			return Plan{}, apperr.New(apperr.Integrity, "archive.patch", pathName, "unsupported patch operation: non-regular file")
		}
		plan.Entries = append(plan.Entries, PlanEntry{
			Target: target,
			File:   f,
			Mode:   fi.Mode().Perm(),
		})
	}
	return plan, nil
}

func classifyPatchFile(patchFile string, f *gitdiff.File) error {
	if f.OldName == "/dev/null" || f.NewName == "/dev/null" {
		return unsupportedPatchOp(patchFile, "create or delete path")
	}
	if f.IsNew || f.IsDelete || f.IsRename || f.IsCopy {
		return unsupportedPatchOp(patchFile, "create, delete, rename, or copy")
	}
	old := filepath.ToSlash(strings.TrimSpace(f.OldName))
	newName := filepath.ToSlash(strings.TrimSpace(f.NewName))
	if old != "" && newName != "" && old != newName {
		return unsupportedPatchOp(patchFile, "rename")
	}
	if f.IsBinary || f.BinaryFragment != nil {
		return unsupportedPatchOp(patchFile, "binary patch")
	}
	if len(f.TextFragments) == 0 && f.OldMode != f.NewMode {
		return unsupportedPatchOp(patchFile, "mode-only change")
	}
	if len(f.TextFragments) == 0 {
		return unsupportedPatchOp(patchFile, "empty change")
	}
	return nil
}

func patchTargetName(f *gitdiff.File) string {
	name := strings.TrimSpace(f.OldName)
	if name == "" {
		name = strings.TrimSpace(f.NewName)
	}
	return name
}

func unsupportedPatchOp(patchFile, detail string) error {
	return apperr.New(apperr.Integrity, "archive.patch", patchFile,
		fmt.Sprintf("unsupported patch operation: %s", detail))
}

// ApplyPlan executes a preflight-validated plan.
func ApplyPlan(ctx context.Context, plan Plan) error {
	for _, e := range plan.Entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := recheckPatchTarget(plan.Root, e.Target); err != nil {
			return err
		}
		orig, err := os.ReadFile(e.Target)
		if err != nil {
			return apperr.Wrap(apperr.IO, "archive.patch", e.Target, err)
		}
		var out bytes.Buffer
		if err := gitdiff.Apply(&out, bytes.NewReader(orig), e.File); err != nil {
			return apperr.Wrap(apperr.Integrity, "archive.patch", e.Target, err)
		}
		if err := checkPatchExpansion(len(orig), out.Len()); err != nil {
			return err
		}
		if err := recheckPatchTarget(plan.Root, e.Target); err != nil {
			return err
		}
		if err := os.WriteFile(e.Target, out.Bytes(), e.Mode); err != nil {
			return apperr.Wrap(apperr.IO, "archive.patch", e.Target, err)
		}
	}
	return nil
}

// ApplyUnifiedPatch applies a unified diff patch file to all files under destDir.
func ApplyUnifiedPatch(ctx context.Context, destDir, patchPath string) error {
	plan, err := PreflightPlan(ctx, patchPath, destDir)
	if err != nil {
		return err
	}
	return ApplyPlan(ctx, plan)
}

func readAllLimited(r io.Reader, limit int64) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, r, limit); err != nil && err != io.EOF {
		return nil, err
	}
	if buf.Len() >= int(limit) {
		extra, err := io.ReadAll(io.LimitReader(r, 1))
		if len(extra) > 0 {
			return nil, fmt.Errorf("file exceeds limit")
		}
		if err != nil && err != io.EOF {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func openFileLimited(path string, limit int64) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if st.Size() > limit {
		_ = f.Close()
		return nil, fmt.Errorf("file exceeds limit")
	}
	return f, nil
}
