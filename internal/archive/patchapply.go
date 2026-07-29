package archive

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/mewisme/mew/internal/apperr"
)

// ApplyUnifiedPatch applies a unified diff patch file to all files under destDir.
func ApplyUnifiedPatch(ctx context.Context, destDir, patchPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(patchPath)
	if err != nil {
		return apperr.Wrap(apperr.IO, "archive.patch", patchPath, err)
	}
	norm := strings.ReplaceAll(string(data), "\r\n", "\n")
	files, _, err := gitdiff.Parse(strings.NewReader(norm))
	if err != nil {
		return apperr.Wrap(apperr.IO, "archive.patch", patchPath, err)
	}
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		target := filepath.Join(destDir, filepath.FromSlash(f.OldName))
		if f.OldName == "/dev/null" || f.OldName == "" {
			target = filepath.Join(destDir, filepath.FromSlash(f.NewName))
		}
		orig, err := os.ReadFile(target)
		if err != nil {
			return apperr.Wrap(apperr.IO, "archive.patch", target, fmt.Errorf("patch target %q: %w", f.OldName, err))
		}
		var out bytes.Buffer
		if err := gitdiff.Apply(&out, bytes.NewReader(orig), f); err != nil {
			return apperr.Wrap(apperr.IO, "archive.patch", target, err)
		}
		if err := os.WriteFile(target, out.Bytes(), 0o644); err != nil {
			return apperr.Wrap(apperr.IO, "archive.patch", target, err)
		}
	}
	return nil
}
