// Package presentation resolves CLI output modes and owns the presentation controller
// boundary between command wiring and diagnostics reporters.
package presentation

import "fmt"

// OutputMode is the canonical human/machine output mode for one invocation.
type OutputMode string

const (
	OutputRich   OutputMode = "rich"
	OutputPlain  OutputMode = "plain"
	OutputJSON   OutputMode = "json"
	OutputNDJSON OutputMode = "ndjson"
	OutputSilent OutputMode = "silent"
)

// Structured reports machine-only output modes.
func (m OutputMode) Structured() bool {
	return m == OutputJSON || m == OutputNDJSON
}

// LogLevel controls diagnostic verbosity.
type LogLevel string

const (
	LogError LogLevel = "error"
	LogWarn  LogLevel = "warn"
	LogInfo  LogLevel = "info"
	LogDebug LogLevel = "debug"
)

// Outcome is the command result passed to Controller.Close.
type Outcome struct {
	Err error
}

// ConflictError reports incompatible explicit presentation flags.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	if e == nil || e.Message == "" {
		return "conflicting presentation flags"
	}
	return e.Message
}

// InvalidModeError reports an unknown output or policy value.
type InvalidModeError struct {
	Field string
	Value string
}

func (e *InvalidModeError) Error() string {
	if e == nil {
		return "invalid presentation mode"
	}
	return fmt.Sprintf("invalid %s: %q", e.Field, e.Value)
}
