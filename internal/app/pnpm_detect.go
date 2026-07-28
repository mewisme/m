package app

import (
	"os"

	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func lockHintsFromProject(proj *project.Project) lockfile.ProjectHints {
	h := project.LockfileHints(proj)
	return lockfile.ProjectHints{
		PackageManager: h.PackageManager,
		DevEnginesPM:   h.DevEnginesPM,
	}
}

func detectPnpmLock(prior []byte, proj *project.Project, explicitMajor int) (lockfile.Detection, error) {
	return detectPnpmLockBytes(prior, lockHintsFromProject(proj), explicitMajor)
}

func detectPnpmLockBytes(prior []byte, hints lockfile.ProjectHints, explicitMajor int) (lockfile.Detection, error) {
	det, err := lockfile.DetectPnpmForProject(prior, hints, explicitMajor)
	if err != nil {
		return det, err
	}
	if explicitMajor != 0 {
		det.ExplicitMajor = true
	}
	return det, nil
}

func validatePnpmLockBeforeTxn(proj *project.Project) error {
	if proj == nil || proj.Identity != project.IdentityPNPM {
		return nil
	}
	if lockfile.PnpmValidateSupported == nil {
		return nil
	}
	prior, err := project.ReadLockfileBytes(proj.Root, proj.Identity)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return lockfile.PnpmValidateSupported(prior)
}
