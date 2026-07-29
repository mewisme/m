package app

import (
	"os"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/npm"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func readExtLockPrior(proj *project.Project) ([]byte, error) {
	prior, err := project.ReadLockfileBytes(proj.Root, proj.Identity)
	if err == nil {
		return prior, nil
	}
	if isLockNotFound(err) && (proj.Identity == project.IdentityNPM || proj.Identity == project.IdentityBun) {
		return nil, nil
	}
	return nil, err
}

func isLockNotFound(err error) bool {
	if os.IsNotExist(err) {
		return true
	}
	return apperr.CodeOf(err) == apperr.NotFound
}

func detectNpmLock(prior []byte) (lockfile.Detection, error) {
	if len(prior) == 0 {
		return lockfile.Detection{
			Format: npm.FormatV3, ProducerMajor: 3, Confidence: lockfile.DetectionCertain,
			Evidence: []string{"greenfield"},
		}, nil
	}
	doc, err := npm.Decode(prior)
	if err != nil {
		return lockfile.Detection{}, err
	}
	if err := npm.ValidateSupported(doc); err != nil {
		return lockfile.Detection{}, err
	}
	return npm.DetectFromDocument(doc), nil
}

func validateNpmLockBeforeTxn(proj *project.Project) error {
	if proj == nil || proj.Identity != project.IdentityNPM {
		return nil
	}
	prior, err := project.ReadLockfileBytes(proj.Root, proj.Identity)
	if err != nil {
		if isLockNotFound(err) {
			return nil
		}
		return err
	}
	doc, err := npm.Decode(prior)
	if err != nil {
		return err
	}
	return npm.ValidateSupported(doc)
}
