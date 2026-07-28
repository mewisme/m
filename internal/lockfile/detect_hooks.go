package lockfile

// PnpmStructureDetect classifies parsed pnpm lock bytes using compat policy evidence.
// Registered by internal/compat/pnpm at init to avoid an import cycle.
var PnpmStructureDetect func(data []byte) (Detection, bool)

// RegisterPnpmStructureDetect installs policy-owned structural detection for pnpm locks.
func RegisterPnpmStructureDetect(fn func(data []byte) (Detection, bool)) {
	PnpmStructureDetect = fn
}
