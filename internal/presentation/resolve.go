package presentation

import (
	"strings"
)

// Input collects raw CLI, environment, and config values before resolution.
type Input struct {
	OutputFlag        string
	ReporterFlag      string
	ColorFlag         string
	NoColor           bool
	ProgressFlag      string
	UnicodeFlag       string
	InteractiveFlag   string
	LogLevelFlag      string
	NoSummary         bool
	Accessible        bool
	LegacyFlag        bool
	Debug             bool
	Unsafe            bool
	Env               map[string]string
	Config            map[string]string
	MandateStructured OutputMode
}

// Resolve computes immutable presentation options from input and capabilities.
func Resolve(input Input, caps Capabilities) (ResolvedOptions, error) {
	if err := checkConflicts(input); err != nil {
		return ResolvedOptions{}, err
	}

	legacy := input.LegacyFlag || envTruthy(input.Env, "MEW_PRESENTATION", "legacy")

	requested, sourceReporter, err := resolveRequestedOutput(input)
	if err != nil {
		return ResolvedOptions{}, err
	}
	if legacy && requested == OutputAuto {
		requested = OutputPlain
	}

	effective := requested
	downgraded := false
	if effective == OutputAuto {
		if caps.RichEligible(input.Accessible) && !legacy {
			effective = OutputRich
		} else {
			effective = OutputPlain
		}
	}
	if input.MandateStructured != "" && !effective.Structured() {
		effective = input.MandateStructured
	}

	if effective == OutputRich && !caps.RichEligible(input.Accessible) {
		if requested == OutputRich {
			return ResolvedOptions{}, &RichUnsupportedError{Reason: "stderr is not an interactive terminal"}
		}
		effective = OutputPlain
		downgraded = true
	}

	// Explicit --color=always overrides NO_COLOR / --no-color.
	cliColorAlways := strings.EqualFold(strings.TrimSpace(input.ColorFlag), "always")
	forceNeverColor := !cliColorAlways && (input.NoColor || caps.NoColorEnv || envTruthy(input.Env, "NO_COLOR", ""))
	color, err := resolveTriState("color", firstNonEmpty(
		input.ColorFlag,
		envString(input.Env, "MEW_COLOR"),
		configString(input.Config, "ui.color"),
	), forceNeverColor)
	if err != nil {
		return ResolvedOptions{}, err
	}
	progress, err := resolveTriState("progress", firstNonEmpty(
		input.ProgressFlag,
		envString(input.Env, "MEW_PROGRESS"),
		configString(input.Config, "ui.progress"),
	), false)
	if err != nil {
		return ResolvedOptions{}, err
	}
	unicode, err := resolveTriState("unicode", firstNonEmpty(
		input.UnicodeFlag,
		envString(input.Env, "MEW_UNICODE"),
		configString(input.Config, "ui.unicode"),
	), false)
	if err != nil {
		return ResolvedOptions{}, err
	}
	interactive, err := resolveTriState("interactive", firstNonEmpty(
		input.InteractiveFlag,
		envString(input.Env, "MEW_INTERACTIVE"),
		configString(input.Config, "ui.interactive"),
	), false)
	if err != nil {
		return ResolvedOptions{}, err
	}

	accessible := input.Accessible || envTruthy(input.Env, "MEW_ACCESSIBLE", "1") ||
		configBool(input.Config, "ui.accessible")
	if accessible && effective == OutputRich {
		effective = OutputPlain
	}

	logLevel, err := resolveLogLevel(firstNonEmpty(
		input.LogLevelFlag,
		envString(input.Env, "MEW_LOG_LEVEL"),
		configString(input.Config, "log.level"),
	), input.Debug)
	if err != nil {
		return ResolvedOptions{}, err
	}

	summary := !input.NoSummary
	if _, ok := input.Config["ui.summary"]; ok {
		summary = configBool(input.Config, "ui.summary")
	}

	_ = sourceReporter // reserved for diagnostics provenance

	width := caps.Width
	if width <= 0 {
		width = defaultTermWidth
	}
	width = ClampWidth(width)

	return ResolvedOptions{
		RequestedOutput: requested,
		EffectiveOutput: effective,
		Color:           color,
		Progress:        progress,
		Unicode:         unicode,
		Interactive:     interactive,
		Accessible:      accessible,
		Summary:         summary,
		Theme:           configString(input.Config, "ui.theme"),
		LogLevel:        logLevel,
		Debug:           logLevel == LogDebug,
		Unsafe:          input.Unsafe,
		Legacy:          legacy,
		DowngradedRich:  downgraded,
		TermWidth:       width,
	}, nil
}

func resolveRequestedOutput(input Input) (OutputMode, bool, error) {
	fromCLI := strings.TrimSpace(input.OutputFlag)
	fromReporter := strings.TrimSpace(input.ReporterFlag)
	fromEnv := firstNonEmpty(
		envString(input.Env, "MEW_OUTPUT"),
		envString(input.Env, "MEW_LOG_FORMAT"),
	)
	fromConfig := configString(input.Config, "ui.output")

	if fromCLI != "" {
		mode, err := parseOutputMode(fromCLI)
		if err != nil {
			return "", false, err
		}
		return mode, false, nil
	}
	if fromReporter != "" {
		mode, err := reporterToOutput(fromReporter)
		if err != nil {
			return "", true, err
		}
		return mode, true, nil
	}
	if fromEnv != "" {
		mode, err := parseOutputOrReporter(fromEnv)
		if err != nil {
			return "", false, err
		}
		return mode, strings.EqualFold(strings.TrimSpace(envString(input.Env, "MEW_LOG_FORMAT")), fromEnv), nil
	}
	if fromConfig != "" {
		mode, err := parseOutputMode(fromConfig)
		if err != nil {
			return "", false, err
		}
		return mode, false, nil
	}
	return OutputAuto, false, nil
}

func checkConflicts(input Input) error {
	cliOut := strings.TrimSpace(input.OutputFlag)
	cliRep := strings.TrimSpace(input.ReporterFlag)
	if cliOut != "" && cliRep != "" {
		outMode, err := parseOutputMode(cliOut)
		if err != nil {
			return err
		}
		repMode, err := reporterToOutput(cliRep)
		if err != nil {
			return err
		}
		if concreteConflict(outMode, repMode) {
			return &ConflictError{Message: "conflicting --output and --reporter flags"}
		}
	}
	envOut := envString(input.Env, "MEW_OUTPUT")
	envRep := envString(input.Env, "MEW_LOG_FORMAT")
	if envOut != "" && envRep != "" {
		outMode, err := parseOutputMode(envOut)
		if err != nil {
			return err
		}
		repMode, err := reporterToOutput(envRep)
		if err != nil {
			return err
		}
		if concreteConflict(outMode, repMode) {
			return &ConflictError{Message: "conflicting MEW_OUTPUT and MEW_LOG_FORMAT"}
		}
	}
	return nil
}

func concreteConflict(a, b OutputMode) bool {
	if a == OutputAuto || b == OutputAuto {
		return false
	}
	return a != b
}

func parseOutputOrReporter(v string) (OutputMode, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return OutputAuto, nil
	}
	if mode, err := parseOutputMode(v); err == nil {
		return mode, nil
	}
	return reporterToOutput(v)
}

func parseOutputMode(v string) (OutputMode, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "auto":
		return OutputAuto, nil
	case "rich":
		return OutputRich, nil
	case "plain":
		return OutputPlain, nil
	case "json":
		return OutputJSON, nil
	case "ndjson":
		return OutputNDJSON, nil
	case "silent":
		return OutputSilent, nil
	case "default", "human":
		return OutputAuto, nil
	default:
		return "", &InvalidModeError{Field: "output", Value: v}
	}
}

func reporterToOutput(v string) (OutputMode, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "default", "human":
		return OutputAuto, nil
	case "json":
		return OutputJSON, nil
	case "ndjson":
		return OutputNDJSON, nil
	case "silent":
		return OutputSilent, nil
	default:
		return "", &InvalidModeError{Field: "reporter", Value: v}
	}
}

func resolveTriState(field, raw string, forceNever bool) (TriState, error) {
	if forceNever {
		return TriNever, nil
	}
	if raw == "" {
		return TriAuto, nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "auto":
		return TriAuto, nil
	case "always":
		return TriAlways, nil
	case "never":
		return TriNever, nil
	default:
		return "", &InvalidModeError{Field: field, Value: raw}
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

func envString(env map[string]string, key string) string {
	if env == nil {
		return ""
	}
	return strings.TrimSpace(env[key])
}

func envTruthy(env map[string]string, key, want string) bool {
	v := envString(env, key)
	if v == "" {
		return false
	}
	if want == "" {
		return isTruthy(v)
	}
	return strings.EqualFold(v, want)
}

func configString(cfg map[string]string, key string) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg[key])
}

func configBool(cfg map[string]string, key string) bool {
	v := strings.ToLower(configString(cfg, key))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Structured conflicts with command-local --json result encoding.
func StructuredConflictsWithCommandJSON(opts ResolvedOptions, commandJSON bool) error {
	if !commandJSON || !opts.Structured() {
		return nil
	}
	return &ConflictError{Message: "command --json conflicts with structured --output mode"}
}
