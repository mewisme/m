package pnpm

import (
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func init() {
	lockfile.RegisterExtAdapter(project.IdentityPNPM, Adapter{})
	lockfile.RegisterPnpmStructureDetect(func(data []byte) (lockfile.Detection, bool) {
		doc, err := Decode(data)
		if err != nil {
			return lockfile.Detection{}, false
		}
		return DetectFromDocument(doc)
	})
}
