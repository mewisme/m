package presentation

import (
	"fmt"
	"strings"
)

// MarkdownTheme selects a Glamour Markdown rendering theme for m help and topic output.
type MarkdownTheme string

const (
	MarkdownThemeDark       MarkdownTheme = "dark"
	MarkdownThemeLight      MarkdownTheme = "light"
	MarkdownThemeDracula    MarkdownTheme = "dracula"
	MarkdownThemeTokyoNight MarkdownTheme = "tokyo-night"
	MarkdownThemeNoTTY      MarkdownTheme = "notty"

	// MarkdownThemeASCII is internal-only and disables all Glamour styling for accessibility.
	markdownThemeASCII MarkdownTheme = "ascii"
)

// DefaultMarkdownTheme is the theme used when no explicit configuration is present.
const DefaultMarkdownTheme = MarkdownThemeDark

// ParseMarkdownTheme validates and normalizes a raw string into a MarkdownTheme.
// Empty and whitespace-only strings produce the default (dark).
func ParseMarkdownTheme(raw string) (MarkdownTheme, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultMarkdownTheme, nil
	}
	switch strings.ToLower(raw) {
	case "dark":
		return MarkdownThemeDark, nil
	case "light":
		return MarkdownThemeLight, nil
	case "dracula":
		return MarkdownThemeDracula, nil
	case "tokyo-night":
		return MarkdownThemeTokyoNight, nil
	case "notty":
		return MarkdownThemeNoTTY, nil
	default:
		return "", fmt.Errorf("unknown markdown theme %q; want dark|light|dracula|tokyo-night|notty", raw)
	}
}

// SupportedMarkdownThemes returns the user-facing theme list in display order.
func SupportedMarkdownThemes() []MarkdownTheme {
	return []MarkdownTheme{
		MarkdownThemeDark,
		MarkdownThemeLight,
		MarkdownThemeDracula,
		MarkdownThemeTokyoNight,
		MarkdownThemeNoTTY,
	}
}

// ResolveMarkdownTheme applies no-color and ascii overrides to a configured theme.
// An empty configured value falls back to the default (dark).
func ResolveMarkdownTheme(configured MarkdownTheme, noColor bool, ascii bool) MarkdownTheme {
	if ascii {
		return markdownThemeASCII
	}
	if noColor {
		return MarkdownThemeNoTTY
	}
	if configured == "" {
		return DefaultMarkdownTheme
	}
	return configured
}

// GlamourStyle maps a MarkdownTheme to the Glamour WithStandardStyle name.
// ascii returns notty (no-op for Glamour since ASCII mode skips Glamour entirely).
func (t MarkdownTheme) GlamourStyle() string {
	switch t {
	case MarkdownThemeLight:
		return "light"
	case MarkdownThemeTokyoNight:
		return "tokyo-night"
	case MarkdownThemeDracula:
		return "dracula"
	case MarkdownThemeNoTTY, markdownThemeASCII:
		return "notty"
	default:
		return "dark"
	}
}
