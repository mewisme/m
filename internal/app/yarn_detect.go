package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/yarn"
	"github.com/mewisme/mew/internal/compat/yarn/berry"
	"github.com/mewisme/mew/internal/compat/yarn/classic"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func detectYarnLock(prior []byte, root string) (lockfile.Detection, error) {
	if len(prior) == 0 {
		return lockfile.Detection{
			Format: greenfieldYarnFormat(root), ProducerMajor: greenfieldYarnMajor(root),
			Confidence: lockfile.DetectionInferred, Evidence: []string{"greenfield"},
		}, nil
	}
	switch yarn.DetectVariant(prior, root) {
	case yarn.VariantBerryPnP:
		return lockfile.Detection{
			Format: berry.FormatBerryPnP, ProducerMajor: 4, Confidence: lockfile.DetectionCertain,
			Evidence: []string{"__metadata", "pnp"},
		}, nil
	case yarn.VariantBerryNM:
		return lockfile.Detection{
			Format: berry.FormatBerryNM, ProducerMajor: 4, Confidence: lockfile.DetectionCertain,
			Evidence: []string{"__metadata", "node-modules"},
		}, nil
	default:
		return lockfile.Detection{
			Format: classic.FormatClassic, ProducerMajor: 1, Confidence: lockfile.DetectionCertain,
			Evidence: []string{"classic"},
		}, nil
	}
}

func greenfieldYarnFormat(root string) string {
	if yarn.HasPnPLinker(root) {
		return berry.FormatBerryPnP
	}
	return classic.FormatClassic
}

func greenfieldYarnMajor(root string) int {
	if yarn.HasPnPLinker(root) {
		return 4
	}
	return 1
}

func validateYarnLockBeforeTxn(proj *project.Project) error {
	if proj == nil || proj.Identity != project.IdentityYarn {
		return nil
	}
	lockPath := filepath.Join(proj.Root, "yarn.lock")
	if _, err := os.Stat(lockPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "yarn.validate", lockPath, err)
	}
	_, _, err := yarn.Adapter{}.ReadWithExtensions(context.Background(), lockPath)
	return err
}

func gateYarnPnPInstall(proj *project.Project) error {
	if proj == nil || proj.Identity != project.IdentityYarn {
		return nil
	}
	lockPath := filepath.Join(proj.Root, "yarn.lock")
	prior, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "install.gate", lockPath, err)
	}
	if !yarn.IsPnPProject(proj.Root, prior) {
		return nil
	}
	return apperr.New(apperr.PNPUnsupported, "install", "yarn.lock",
		"Yarn Berry PnP install is not supported; use node-modules linker or migrate to m.lock (see docs/yarn-lockfile.md)")
}
