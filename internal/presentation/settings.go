package presentation

import "strings"

// ThemeMode selects a semantic palette.
type ThemeMode string

const (
	ThemeAuto       ThemeMode = "auto"
	ThemeLight      ThemeMode = "light"
	ThemeDark       ThemeMode = "dark"
	ThemeAccessible ThemeMode = "accessible"
	ThemeNone       ThemeMode = "none"
)

// DarkModeDetector wraps OS dark-mode detection.
// The presentation package does not import platform-specific code directly.
type DarkModeDetector interface {
	IsDarkMode() (bool, error)
}

// EffectiveSettings is the immutable rendering policy derived once per invocation.
type EffectiveSettings struct {
	UseColor       bool
	UseUnicode     bool
	UseProgress    bool
	UseInteractive bool
	ThemeMode      ThemeMode
	Width          int
	Accessible     bool
	Symbols        Symbols
	BinaryName     string // invoked binary name: "m", "mew", "mx", or "mewx"
}

// BinName returns the binary name for user-facing command references, defaulting to "m".
func (s EffectiveSettings) BinName() string {
	if s.BinaryName != "" {
		return s.BinaryName
	}
	return "m"
}

// Effective derives rendering settings from resolved options and capabilities.
// dark is the optional OS dark-mode detector; nil means detection is unavailable.
func Effective(resolved ResolvedOptions, caps Capabilities, dark DarkModeDetector) EffectiveSettings {
	width := resolved.TermWidth
	if width <= 0 {
		width = caps.Width
	}
	width = ClampWidth(width)

	useColor := resolveUseColor(resolved, caps)
	useUnicode := resolved.Unicode
	useProgress := resolved.Progress && caps.StderrTTY && !caps.CI && !caps.DumbTerminal && !resolved.Accessible && !caps.ScreenReader
	useInteractive := caps.Interactive
	themeMode := resolveThemeMode(resolved, caps, useColor, dark)

	return EffectiveSettings{
		UseColor:       useColor,
		UseUnicode:     useUnicode,
		UseProgress:    useProgress,
		UseInteractive: useInteractive,
		ThemeMode:      themeMode,
		Width:          width,
		Accessible:     resolved.Accessible || caps.ScreenReader,
		Symbols:        SelectSymbols(useUnicode),
	}
}

// resolveUseColor decides whether static rich styling (Lip Gloss ANSI) is allowed.
// Explicit --output=plain, accessible, structured, silent, and --no-color never get color.
func resolveUseColor(resolved ResolvedOptions, caps Capabilities) bool {
	if resolved.Accessible || caps.ScreenReader {
		return false
	}
	if !resolved.Color {
		return false
	}
	if resolved.Structured() || resolved.Output == OutputSilent {
		return false
	}
	if resolved.Output == OutputPlain {
		return false
	}
	if resolved.Output == OutputRich {
		return caps.StdoutTTY && !caps.DumbTerminal && caps.SupportsColor()
	}
	return false
}

func resolveThemeMode(resolved ResolvedOptions, caps Capabilities, useColor bool, dark DarkModeDetector) ThemeMode {
	if !useColor {
		return ThemeNone
	}
	if resolved.Accessible || caps.ScreenReader {
		return ThemeAccessible
	}
	return ResolveTheme(resolved.Theme, dark)
}

// ResolveTheme maps a configured ui.theme value and detector result to a ThemeMode.
//
// Precedence:
//
//	ui.theme=light  → ThemeLight
//	ui.theme=dark   → ThemeDark
//	ui.theme=auto   → detector result → ThemeDark or ThemeLight; fallback ThemeLight on error/unknown
//	empty/missing   → same as auto
//
// ThemeAccessible and ThemeNone are handled by callers (resolveThemeMode) based on
// accessibility and color policy — they are not valid ui.theme values.
func ResolveTheme(configured string, dark DarkModeDetector) ThemeMode {
	switch strings.ToLower(strings.TrimSpace(configured)) {
	case "light":
		return ThemeLight
	case "dark":
		return ThemeDark
	default: // "auto", "", anything else → detect
		if dark == nil {
			return ThemeLight
		}
		isDark, err := dark.IsDarkMode()
		if err != nil || !isDark {
			return ThemeLight
		}
		return ThemeDark
	}
}
