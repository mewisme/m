package presentation

import (
	"strings"
)

// Input collects raw CLI flag values before resolution.
// Presentation env vars and config keys are no longer read.
type Input struct {
	OutputFlag    string
	NoColor       bool
	ASCII         bool
	NoProgress    bool
	Accessible    bool
	NoSummary     bool
	LogLevelFlag  string
	Debug         bool
	Unsafe        bool
	MarkdownTheme string
}

// Resolve computes immutable presentation options from explicit CLI flags.
// Rich is always the default; env and config no longer influence presentation.
func Resolve(input Input) (ResolvedOptions, error) {
	output, err := parseOutputMode(input.OutputFlag)
	if err != nil {
		return ResolvedOptions{}, err
	}

	logLevel, err := resolveLogLevel(input.LogLevelFlag, input.Debug)
	if err != nil {
		return ResolvedOptions{}, err
	}

	color := !input.NoColor && !output.Structured() && output != OutputSilent && output != OutputPlain
	progress := !input.NoProgress && output == OutputRich
	unicode := !input.ASCII
	summary := !input.NoSummary

	return ResolvedOptions{
		Output:        output,
		Color:         color,
		Progress:      progress,
		Unicode:       unicode,
		Accessible:    input.Accessible,
		Summary:       summary,
		Theme:         "",
		MarkdownTheme: input.MarkdownTheme,
		LogLevel:      logLevel,
		Debug:         logLevel == LogDebug,
		Unsafe:        input.Unsafe,
	}, nil
}

func parseOutputMode(v string) (OutputMode, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "rich":
		return OutputRich, nil
	case "plain":
		return OutputPlain, nil
	case "json":
		return OutputJSON, nil
	case "ndjson":
		return OutputNDJSON, nil
	case "silent":
		return OutputSilent, nil
	case "auto", "default", "human":
		return "", &InvalidModeError{
			Field: "output",
			Value: v + " (not accepted; use rich, plain, json, ndjson, or silent)",
		}
	default:
		return "", &InvalidModeError{Field: "output", Value: v}
	}
}

func resolveLogLevel(raw string, debugFlag bool) (LogLevel, error) {
	if debugFlag {
		return LogDebug, nil
	}
	if raw == "" {
		return LogError, nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "error":
		return LogError, nil
	case "warn":
		return LogWarn, nil
	case "info":
		return LogInfo, nil
	case "debug":
		return LogDebug, nil
	default:
		return "", &InvalidModeError{Field: "log-level", Value: raw}
	}
}

// StructuredConflictsWithCommandJSON reports when command-local --json conflicts with structured output.
func StructuredConflictsWithCommandJSON(opts ResolvedOptions, commandJSON bool) error {
	if !commandJSON || !opts.Structured() {
		return nil
	}
	return &ConflictError{Message: "command --json conflicts with structured --output mode"}
}
