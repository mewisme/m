package pnpm

import (
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func init() {
	lockfile.RegisterExtAdapter(project.IdentityPNPM, Adapter{})
}
