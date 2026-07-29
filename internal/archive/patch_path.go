package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// PatchPathError describes a rejected patch target path.
type PatchPathError struct {
	PatchFile   string
	PackageRoot string
	Original    string
	Normalized  string
	Reason      string
}

func (e *PatchPathError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("patch path %q: %s", e.Original, e.Reason)
}

func newPatchPathError(patchFile, root, original, normalized, reason string) error {
	ppe := &PatchPathError{
		PatchFile:   patchFile,
		PackageRoot: root,
		Original:    original,
		Normalized:  normalized,
		Reason:      reason,
	}
	return apperr.Wrap(apperr.Integrity, "archive.patch.path", original, ppe)
}

// resolvePatchTarget maps a patch file path inside package root to an absolute path.
func resolvePatchTarget(patchFile, root, name string) (string, error) {
	original := name
	name = filepath.ToSlash(strings.TrimSpace(name))
	if name == "" {
		return "", newPatchPathError(patchFile, root, original, name, "empty path")
	}
	if name == "/dev/null" {
		return "", newPatchPathError(patchFile, root, original, name, "dev null path")
	}
	if strings.HasPrefix(name, "//") || strings.HasPrefix(name, "\\\\") {
		return "", newPatchPathError(patchFile, root, original, name, "unc path")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "archive.patch.path", root, err)
	}
	target, err := safeJoin(rootAbs, name)
	if err != nil {
		return "", newPatchPathError(patchFile, root, original, name, pathReason(err))
	}
	if err := fsx.GuardAncestors(rootAbs, target); err != nil {
		return "", newPatchPathError(patchFile, root, original, name, pathReason(err))
	}
	return target, nil
}

// recheckPatchTarget re-validates containment before a filesystem mutation.
func recheckPatchTarget(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return apperr.Wrap(apperr.IO, "archive.patch.path", root, err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return apperr.Wrap(apperr.IO, "archive.patch.path", target, err)
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return apperr.New(apperr.Integrity, "archive.patch.path", target, "path escapes package root")
	}
	if runtime.GOOS == "windows" {
		if !sameVolume(rootAbs, targetAbs) {
			return apperr.New(apperr.Integrity, "archive.patch.path", target, "cross-volume path")
		}
	}
	return fsx.GuardAncestors(rootAbs, targetAbs)
}

func sameVolume(a, b string) bool {
	volA := volumeName(a)
	volB := volumeName(b)
	return volA == volB
}

// ResolvePatchTargetForTest exposes path resolution for external tests.
func ResolvePatchTargetForTest(patchFile, root, name string) (string, error) {
	return resolvePatchTarget(patchFile, root, name)
}

func volumeName(path string) string {
	if len(path) < 2 || path[1] != ':' {
		return ""
	}
	return strings.ToUpper(path[:2])
}

func pathReason(err error) string {
	if err == nil {
		return "invalid path"
	}
	var ae *apperr.Error
	if errors.As(err, &ae) && ae.Message != "" {
		return ae.Message
	}
	return err.Error()
}
