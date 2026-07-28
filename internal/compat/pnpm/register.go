package pnpm

import (
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func init() {
	lockfile.RegisterExtAdapter(project.IdentityPNPM, Adapter{})
	lockfile.RegisterPnpmValidateSupported(func(data []byte) error {
		doc, err := Decode(data)
		if err != nil {
			return err
		}
		return ValidateSupportedPnpm(doc)
	})
	lockfile.RegisterPnpmStructureDetect(func(data []byte) (lockfile.Detection, bool) {
		doc, err := Decode(data)
		if err != nil {
			return lockfile.Detection{}, false
		}
		return DetectFromDocument(doc)
	})
}
