package archive

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// WriteTreePatch writes a unified diff of regular files in editedRoot against originalRoot.
// Only in-place modifications are emitted; create/delete/rename paths are rejected.
func WriteTreePatch(ctx context.Context, originalRoot, editedRoot string) ([]byte, error) {
	origAbs, err := filepath.Abs(originalRoot)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "archive.diff", originalRoot, err)
	}
	editAbs, err := filepath.Abs(editedRoot)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "archive.diff", editedRoot, err)
	}
	var out bytes.Buffer
	seen := map[string]struct{}{}
	err = filepath.WalkDir(editAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return apperr.New(apperr.Integrity, "archive.diff", path, "non-regular file in patch tree")
		}
		rel, err := filepath.Rel(editAbs, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, dup := seen[rel]; dup {
			return apperr.New(apperr.Integrity, "archive.diff", rel, "duplicate patch path")
		}
		seen[rel] = struct{}{}
		oldPath := filepath.Join(origAbs, filepath.FromSlash(rel))
		oldData, oldErr := os.ReadFile(oldPath)
		if oldErr != nil {
			if os.IsNotExist(oldErr) {
				return apperr.New(apperr.Integrity, "archive.diff", rel, "patch would create file")
			}
			return apperr.Wrap(apperr.IO, "archive.diff", oldPath, oldErr)
		}
		newData, err := os.ReadFile(path)
		if err != nil {
			return apperr.Wrap(apperr.IO, "archive.diff", path, err)
		}
		if bytes.Equal(oldData, newData) {
			return nil
		}
		chunk, err := unifiedFileDiff(rel, oldData, newData)
		if err != nil {
			return err
		}
		out.Write(chunk)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out.Len() == 0 {
		return nil, apperr.New(apperr.Usage, "archive.diff", editedRoot, "no file changes to patch")
	}
	return out.Bytes(), nil
}

func unifiedFileDiff(rel string, oldData, newData []byte) ([]byte, error) {
	oldLines := splitPatchLines(oldData)
	newLines := splitPatchLines(newData)
	if len(oldLines) == len(newLines) {
		same := true
		for i := range oldLines {
			if oldLines[i] != newLines[i] {
				same = false
				break
			}
		}
		if same {
			return nil, nil
		}
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "diff --git a/%s b/%s\n", rel, rel)
	fmt.Fprintf(&buf, "--- a/%s\n", rel)
	fmt.Fprintf(&buf, "+++ b/%s\n", rel)
	hunks := diffHunks(oldLines, newLines)
	for _, h := range hunks {
		fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n", h.oldStart, h.oldLen, h.newStart, h.newLen)
		for _, line := range h.lines {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes(), nil
}

func splitPatchLines(data []byte) []string {
	norm := strings.ReplaceAll(string(data), "\r\n", "\n")
	if norm == "" {
		return nil
	}
	parts := strings.Split(norm, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

type diffHunk struct {
	oldStart, oldLen int
	newStart, newLen int
	lines            []string
}

func diffHunks(oldLines, newLines []string) []diffHunk {
	// ponytail: single-hunk line diff; upgrade = Myers hunks for large files.
	var h diffHunk
	h.oldStart = 1
	h.newStart = 1
	i, j := 0, 0
	for i < len(oldLines) || j < len(newLines) {
		if i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
			i++
			j++
			continue
		}
		for i < len(oldLines) && (j >= len(newLines) || oldLines[i] != newLines[j]) {
			h.lines = append(h.lines, "-"+oldLines[i])
			h.oldLen++
			i++
		}
		for j < len(newLines) && (i >= len(oldLines) || oldLines[i] != newLines[j]) {
			h.lines = append(h.lines, "+"+newLines[j])
			h.newLen++
			j++
		}
	}
	if h.oldLen == 0 && h.newLen == 0 {
		h.oldLen = len(oldLines)
		h.newLen = len(newLines)
		for _, line := range oldLines {
			h.lines = append(h.lines, "-"+line)
		}
		for _, line := range newLines {
			h.lines = append(h.lines, "+"+line)
		}
	}
	return []diffHunk{h}
}
