package resolver

// ponytail: hard caps for fail-closed greenfield graphs; raise or make configurable when monorepos need more (0022+).
const (
	maxDepth    = 64
	maxPackages = 10_000
)
