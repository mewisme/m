package presentation

import "github.com/mewisme/mew/internal/diagnostics"

// ResolvedOptions is the immutable presentation configuration for one invocation.
type ResolvedOptions struct {
	Output        OutputMode
	Color         bool
	Progress      bool
	Unicode       bool
	Accessible    bool
	Summary       bool
	Theme      string // configured ui.theme value: "auto", "light", "dark", or ""
	LogLevel   LogLevel
	Debug         bool
	Unsafe        bool
	TermWidth     int
	BinaryName    string // "m", "mew", "mx", "mewx"
}

// ReporterFormat maps effective output to diagnostics reporter format names.
func (o ResolvedOptions) ReporterFormat() string {
	switch o.Output {
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
	if o.Color {
		return diagnostics.ColorAlways
	}
	return diagnostics.ColorNever
}

// Structured reports whether machine-only reporter output is active.
func (o ResolvedOptions) Structured() bool {
	switch o.Output {
	case OutputJSON, OutputNDJSON:
		return true
	default:
		return false
	}
}
