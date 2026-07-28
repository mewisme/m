package lifecycle

// ExecutionCapabilities reports what restricted execution enforces for audit.
type ExecutionCapabilities struct {
	PackageCWD          bool `json:"packageCWD"`
	ControlledPATH      bool `json:"controlledPATH"`
	StrippedEnv         bool `json:"strippedEnv"`
	Timeout             bool `json:"timeout"`
	ProcessTreeKill     bool `json:"processTreeKill"`
	FilesystemIsolation bool `json:"filesystemIsolation"`
	NetworkIsolation    bool `json:"networkIsolation"`
}

// DefaultCapabilities returns the honest restricted-execution contract.
func DefaultCapabilities() ExecutionCapabilities {
	return ExecutionCapabilities{
		PackageCWD:          true,
		ControlledPATH:      true,
		StrippedEnv:         true,
		Timeout:             true,
		ProcessTreeKill:     true,
		FilesystemIsolation: false,
		NetworkIsolation:    false,
	}
}
