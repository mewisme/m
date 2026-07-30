package presentation

import "github.com/mewisme/mew/internal/diagnostics"

// ResolvedOptions is the immutable presentation configuration for one invocation.
type ResolvedOptions struct {
	RequestedOutput OutputMode
	EffectiveOutput OutputMode
	Color           TriState
	Progress        TriState
	Unicode         TriState
	Interactive     TriState
	Accessible      bool
	Summary         bool
	Theme           string
	LogLevel        LogLevel
	Debug           bool
	Unsafe          bool
	Legacy          bool
	DowngradedRich  bool
	TermWidth       int
}

// ReporterFormat maps effective output to diagnostics reporter format names.
func (o ResolvedOptions) ReporterFormat() string {
	switch o.EffectiveOutput {
	case OutputJSON:
		return "json"
	case OutputNDJSON:
		return "ndjson"
	case OutputSilent:
		return "silent"
	default:
		return "default"
	}
}

// ColorMode maps resolved color policy to diagnostics.ColorMode.
func (o ResolvedOptions) ColorMode() diagnostics.ColorMode {
	switch o.Color {
	case TriAlways:
		return diagnostics.ColorAlways
	case TriNever:
		return diagnostics.ColorNever
	default:
		return diagnostics.ColorAuto
	}
}

// Structured reports whether machine-only reporter output is active.
func (o ResolvedOptions) Structured() bool {
	switch o.EffectiveOutput {
	case OutputJSON, OutputNDJSON:
		return true
	default:
		return false
	}
}
