package archive

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/mewisme/mew/internal/apperr"
)

// ponytail: fixed patch limits; upgrade = config keys per project policy.

const (
	maxPatchFileSize       = 1 << 20 // 1 MiB
	maxPatchFiles          = 256
	maxPatchHunks          = 4096
	maxPatchPathLen        = 4096
	maxPatchLineLen        = 65536
	maxPatchExpansionRatio = 10
)

func readPatchBytes(ctx context.Context, patchPath string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := readFileLimited(patchPath, maxPatchFileSize+1)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "archive.patch", patchPath, err)
	}
	if len(data) > maxPatchFileSize {
		return nil, apperr.New(apperr.Integrity, "archive.patch", patchPath,
			fmt.Sprintf("patch file exceeds %d bytes", maxPatchFileSize))
	}
	return data, nil
}

func parsePatch(ctx context.Context, patchPath string, data []byte) ([]*gitdiff.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	norm := strings.ReplaceAll(string(data), "\r\n", "\n")
	if err := checkPatchLineLengths(norm); err != nil {
		return nil, err
	}
	files, _, err := gitdiff.Parse(strings.NewReader(norm))
	if err != nil {
		return nil, apperr.Wrap(apperr.Integrity, "archive.patch", patchPath, err)
	}
	if err := checkPatchStructure(patchPath, data, files); err != nil {
		return nil, err
	}
	return files, nil
}

func checkPatchLineLengths(body string) error {
	for _, line := range strings.Split(body, "\n") {
		if len(line) > maxPatchLineLen {
			return apperr.New(apperr.Integrity, "archive.patch", "line",
				fmt.Sprintf("patch line exceeds %d bytes", maxPatchLineLen))
		}
	}
	return nil
}

func checkPatchStructure(patchPath string, data []byte, files []*gitdiff.File) error {
	if len(files) > maxPatchFiles {
		return apperr.New(apperr.Integrity, "archive.patch", patchPath,
			fmt.Sprintf("patch touches more than %d files", maxPatchFiles))
	}
	hunks := 0
	seen := make(map[string]struct{}, len(files))
	for _, f := range files {
		if len(f.OldName) > maxPatchPathLen || len(f.NewName) > maxPatchPathLen {
			return apperr.New(apperr.Integrity, "archive.patch", patchPath,
				fmt.Sprintf("patch path exceeds %d bytes", maxPatchPathLen))
		}
		hunks += len(f.TextFragments)
		key := f.OldName + "\x00" + f.NewName
		if _, dup := seen[key]; dup {
			return apperr.New(apperr.Integrity, "archive.patch", patchPath, "duplicate patch target")
		}
		seen[key] = struct{}{}
	}
	if hunks > maxPatchHunks {
		return apperr.New(apperr.Integrity, "archive.patch", patchPath,
			fmt.Sprintf("patch exceeds %d hunks", maxPatchHunks))
	}
	return nil
}

func checkPatchExpansion(inputSize int, outputSize int) error {
	if inputSize <= 0 {
		return nil
	}
	if outputSize > inputSize*maxPatchExpansionRatio {
		return apperr.New(apperr.Integrity, "archive.patch", "output",
			fmt.Sprintf("patch output exceeds %dx input size", maxPatchExpansionRatio))
	}
	return nil
}

func readFileLimited(path string, limit int64) ([]byte, error) {
	f, err := openFileLimited(path, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return readAllLimited(f, limit)
}
