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
	UseColor                bool
	UseUnicode              bool
	UseProgress             bool
	UseInteractive          bool
	ThemeMode               ThemeMode
	ConfiguredMarkdownTheme MarkdownTheme
	MarkdownTheme           MarkdownTheme
	Width                   int
	Accessible              bool
	Symbols                 Symbols
}

// Effective derives rendering settings from resolved options and capabilities.
func Effective(resolved ResolvedOptions, caps Capabilities) EffectiveSettings {
	width := resolved.TermWidth
	if width <= 0 {
		width = caps.Width
	}
	width = ClampWidth(width)

	useColor := resolveUseColor(resolved, caps)
	useUnicode := resolved.Unicode
	useProgress := resolved.Progress && caps.StderrTTY && !caps.CI && !caps.DumbTerminal && !resolved.Accessible && !caps.ScreenReader
	useInteractive := caps.Interactive
	themeMode := resolveThemeMode(resolved, caps, useColor)
	configuredMarkdown, _ := ParseMarkdownTheme(resolved.MarkdownTheme)
	effectiveMarkdown := ResolveMarkdownTheme(configuredMarkdown, !useColor, !resolved.Unicode)

	return EffectiveSettings{
		UseColor:                useColor,
		UseUnicode:              useUnicode,
		UseProgress:             useProgress,
		UseInteractive:          useInteractive,
		ThemeMode:               themeMode,
		ConfiguredMarkdownTheme: configuredMarkdown,
		MarkdownTheme:           effectiveMarkdown,
		Width:                   width,
		Accessible:              resolved.Accessible || caps.ScreenReader,
		Symbols:                 SelectSymbols(useUnicode),
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

func resolveThemeMode(resolved ResolvedOptions, caps Capabilities, useColor bool) ThemeMode {
	if !useColor {
		return ThemeNone
	}
	return ThemePreference(resolved, caps)
}

// ThemePreference resolves ui.theme (and auto background) without the color gate.
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
