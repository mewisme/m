package lockfile

// PnpmStructureDetect classifies parsed pnpm lock bytes using compat policy evidence.
// Registered by internal/compat/pnpm at init to avoid an import cycle.
var PnpmStructureDetect func(data []byte) (Detection, bool)

// PnpmValidateSupported rejects unsupported pnpm lock layouts before detection/graph work.
// Registered by internal/compat/pnpm at init to avoid an import cycle.
var PnpmValidateSupported func(data []byte) error

// RegisterPnpmStructureDetect installs policy-owned structural detection for pnpm locks.
func RegisterPnpmStructureDetect(fn func(data []byte) (Detection, bool)) {
	PnpmStructureDetect = fn
}

// RegisterPnpmValidateSupported installs unified pnpm support-boundary validation.
func RegisterPnpmValidateSupported(fn func(data []byte) error) {
	PnpmValidateSupported = fn
}
