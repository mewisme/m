package app

import (
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/bun"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func detectBunLock(prior []byte) (lockfile.Detection, error) {
	if len(prior) == 0 {
		return lockfile.Detection{
			Format: bun.FormatV1, ProducerMajor: 1, Confidence: lockfile.DetectionCertain,
			Evidence: []string{"greenfield"},
		}, nil
	}
	doc, err := bun.Decode(prior)
	if err != nil {
		return lockfile.Detection{}, err
	}
	if err := bun.ValidateSupported(doc); err != nil {
		return lockfile.Detection{}, err
	}
	return bun.DetectFromDocument(doc), nil
}

func validateBunLockBeforeTxn(proj *project.Project) error {
	if proj == nil || proj.Identity != project.IdentityBun {
		return nil
	}
	prior, err := project.ReadLockfileBytes(proj.Root, proj.Identity)
	if err != nil {
		if isLockNotFound(err) {
			return nil
		}
		return err
	}
	doc, err := bun.Decode(prior)
	if err != nil {
		return err
	}
	return bun.ValidateSupported(doc)
}

func rejectBunLockbIfPresent(proj *project.Project) error {
	if proj == nil || proj.Identity != project.IdentityBun {
		return nil
	}
	lockbPath := filepath.Join(proj.Root, "bun.lockb")
	data, err := os.ReadFile(lockbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "bun.validate", lockbPath, err)
	}
	if _, decErr := bun.Decode(data); decErr != nil {
		return decErr
	}
	return nil
}
