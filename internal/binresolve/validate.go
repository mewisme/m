package binresolve

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
)

// ValidateCandidate checks shim and target paths before launch.
func ValidateCandidate(cand binmeta.BinCandidate, nodeModules string) error {
	if cand.Command == "" || cand.ShimPath == "" {
		return apperr.New(apperr.Exec, "binresolve.validate", cand.Command, "incomplete candidate")
	}
	binDir := filepath.Join(nodeModules, ".bin")
	shimAbs, err := filepath.Abs(cand.ShimPath)
	if err != nil {
		return apperr.Wrap(apperr.Exec, "binresolve.validate", cand.ShimPath, err)
	}
	binAbs, err := filepath.Abs(binDir)
	if err != nil {
		return apperr.Wrap(apperr.Exec, "binresolve.validate", binDir, err)
	}
	if !pathWithin(binAbs, shimAbs) {
		return apperr.New(apperr.Policy, "binresolve.validate", cand.ShimPath, "shim outside approved .bin directory")
	}
	if runtime.GOOS == "windows" && strings.EqualFold(filepath.Ext(shimAbs), ".ps1") {
		return apperr.New(apperr.Unsupported, "binresolve.validate", cand.Command, "PowerShell shims are not supported")
	}
	st, err := os.Lstat(shimAbs)
	if err != nil {
		return apperr.Wrap(apperr.IO, "binresolve.validate", cand.ShimPath, err)
	}
	if st.Mode()&os.ModeSocket != 0 || st.Mode()&os.ModeDevice != 0 {
		return apperr.New(apperr.Exec, "binresolve.validate", cand.ShimPath, "unsupported shim file type")
	}
	if cand.TargetPath != "" {
		targetAbs, err := filepath.Abs(cand.TargetPath)
		if err != nil {
			return apperr.Wrap(apperr.Exec, "binresolve.validate", cand.TargetPath, err)
		}
		nmAbs, err := filepath.Abs(nodeModules)
		if err != nil {
			return apperr.Wrap(apperr.Exec, "binresolve.validate", nodeModules, err)
		}
		if cand.PackageDir != "" {
			pkgAbs, err := filepath.Abs(cand.PackageDir)
			if err == nil && !pathWithin(pkgAbs, targetAbs) && !pathWithin(nmAbs, targetAbs) {
				return apperr.New(apperr.Policy, "binresolve.validate", cand.TargetPath, "target escapes package area")
			}
		}
	}
	return nil
}

func pathWithin(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	if base == target {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(target, base+sep)
}

// ShimMatchesMetadata reports filesystem/metadata agreement for a record.
func ShimMatchesMetadata(rec binmeta.Record, shimPath string) bool {
	want := rec.MaterializedShim
	if want == "" {
		return false
	}
	binDir := filepath.Dir(shimPath)
	nmDir := filepath.Dir(binDir)
	if !filepath.IsAbs(want) {
		want = filepath.Join(nmDir, want)
	}
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(want), ".cmd") {
		if strings.EqualFold(filepath.Ext(shimPath), ".cmd") {
			want = want + ".cmd"
		}
	}
	a, err1 := filepath.Abs(want)
	b, err2 := filepath.Abs(shimPath)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
