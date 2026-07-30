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

// EffectiveSettings is the immutable rendering policy derived once per invocation.
type EffectiveSettings struct {
	UseColor       bool
	UseUnicode     bool
	UseProgress    bool
	UseInteractive bool
	ThemeMode      ThemeMode
	Width          int
	Accessible     bool
	Legacy         bool
	Symbols        Symbols
}

// Effective derives rendering settings from resolved options and capabilities.
func Effective(resolved ResolvedOptions, caps Capabilities) EffectiveSettings {
	width := resolved.TermWidth
	if width <= 0 {
		width = caps.Width
	}
	width = ClampWidth(width)

	useColor := resolveUseColor(resolved.Color, caps)
	useUnicode := resolveUseUnicode(resolved.Unicode, caps)
	useProgress := resolveUseProgress(resolved, caps)
	useInteractive := resolveUseInteractive(resolved.Interactive, caps)
	themeMode := resolveThemeMode(resolved, caps, useColor)

	return EffectiveSettings{
		UseColor:       useColor,
		UseUnicode:     useUnicode,
		UseProgress:    useProgress,
		UseInteractive: useInteractive,
		ThemeMode:      themeMode,
		Width:          width,
		Accessible:     resolved.Accessible || caps.ScreenReader,
		Legacy:         resolved.Legacy,
		Symbols:        SelectSymbols(useUnicode),
	}
}

func resolveUseColor(color TriState, caps Capabilities) bool {
	switch color {
	case TriAlways:
		return true
	case TriNever:
		return false
	default:
		return caps.StdoutTTY && !caps.DumbTerminal && caps.SupportsColor()
	}
}

func resolveUseUnicode(unicode TriState, caps Capabilities) bool {
	switch unicode {
	case TriAlways:
		return true
	case TriNever:
		return false
	default:
		return caps.Unicode
	}
}

func resolveUseProgress(resolved ResolvedOptions, caps Capabilities) bool {
	switch resolved.Progress {
	case TriAlways:
		return true
	case TriNever:
		return false
	default:
		return resolved.EffectiveOutput == OutputRich &&
			caps.StderrTTY &&
			!caps.CI &&
			!caps.DumbTerminal &&
			!resolved.Accessible &&
			!caps.ScreenReader
	}
}

func resolveUseInteractive(interactive TriState, caps Capabilities) bool {
	switch interactive {
	case TriAlways:
		return caps.StdinTTY
	case TriNever:
		return false
	default:
		return caps.Interactive
	}
}

func resolveThemeMode(resolved ResolvedOptions, caps Capabilities, useColor bool) ThemeMode {
	if !useColor {
		return ThemeNone
	}
	return ThemePreference(resolved, caps)
}

// ThemePreference resolves ui.theme (and auto background) without the color gate.
// Use this for Glamour help styles when ForceColor still renders rich on a non-TTY
// (Effective.ThemeMode is ThemeNone when useColor is false).
func ThemePreference(resolved ResolvedOptions, caps Capabilities) ThemeMode {
	if resolved.Accessible || caps.ScreenReader {
		return ThemeAccessible
	}
	switch strings.ToLower(strings.TrimSpace(resolved.Theme)) {
	case "light":
		return ThemeLight
	case "dark":
		return ThemeDark
	case "accessible":
		return ThemeAccessible
	case "none":
		return ThemeNone
	default: // auto / empty
		if caps.Background == BackgroundDark {
			return ThemeDark
		}
		return ThemeLight
	}
}
