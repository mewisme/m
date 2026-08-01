package presentation

import lipgloss "charm.land/lipgloss/v2"

// Theme holds semantic Lip Gloss styles for one palette.
type Theme struct {
	Primary   lipgloss.Style
	Secondary lipgloss.Style
	Muted     lipgloss.Style
	Strong    lipgloss.Style

	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	Command lipgloss.Style
	Package lipgloss.Style
	Version lipgloss.Style
	Path    lipgloss.Style
	Code    lipgloss.Style
	Number  lipgloss.Style

	Added   lipgloss.Style
	Updated lipgloss.Style
	Removed lipgloss.Style
	Reused  lipgloss.Style

	Header lipgloss.Style
	Label  lipgloss.Style
	Value  lipgloss.Style
}

// NewTheme builds a palette for mode. ThemeNone returns identity styles.
func NewTheme(mode ThemeMode) Theme {
	switch mode {
	case ThemeDark:
		return darkTheme()
	case ThemeAccessible:
		return accessibleTheme()
	case ThemeNone:
		return noneTheme()
	default:
		return lightTheme()
	}
}

func noneTheme() Theme {
	id := lipgloss.NewStyle()
	return Theme{
		Primary: id, Secondary: id, Muted: id, Strong: id,
		Success: id, Warning: id, Error: id, Info: id,
		Command: id, Package: id, Version: id, Path: id, Code: id, Number: id,
		Added: id, Updated: id, Removed: id, Reused: id,
		Header: id, Label: id, Value: id,
	}
}

func lightTheme() Theme {
	return Theme{
		Primary:   fg("#1D4ED8"),
		Secondary: fg("#4B5563"),
		Muted:     fg("#6B7280"),
		Strong:    lipgloss.NewStyle().Bold(true),
		Success:   fg("#15803D"),
		Warning:   fg("#A16207"),
		Error:     fg("#B91C1C"),
		Info:      fg("#1D4ED8"),
		Command:   fg("#1D4ED8"),
		Package:   lipgloss.NewStyle(),
		Version:   lipgloss.NewStyle(),
		Path:      fg("#4B5563"),
		Code:      fg("#6B21A8"),
		Number:    fg("#1F2937"),
		Added:     fg("#15803D"),
		Updated:   fg("#A16207"),
		Removed:   fg("#B91C1C"),
		Reused:    fg("#6B7280"),
		Header:    lipgloss.NewStyle().Bold(true),
		Label:     lipgloss.NewStyle(),
		Value:     lipgloss.NewStyle(),
	}
}

func darkTheme() Theme {
	return Theme{
		Primary:   fg("#93C5FD"),
		Secondary: fg("#9CA3AF"),
		Muted:     fg("#6B7280"),
		Strong:    lipgloss.NewStyle().Bold(true),
		Success:   fg("#4ADE80"),
		Warning:   fg("#FBBF24"),
		Error:     fg("#F87171"),
		Info:      fg("#93C5FD"),
		Command:   fg("#93C5FD"),
		Package:   lipgloss.NewStyle(),
		Version:   lipgloss.NewStyle(),
		Path:      fg("#9CA3AF"),
		Code:      fg("#D8B4FE"),
		Number:    fg("#E5E7EB"),
		Added:     fg("#4ADE80"),
		Updated:   fg("#FBBF24"),
		Removed:   fg("#F87171"),
		Reused:    fg("#6B7280"),
		Header:    lipgloss.NewStyle().Bold(true),
		Label:     lipgloss.NewStyle(),
		Value:     lipgloss.NewStyle(),
	}
}

func accessibleTheme() Theme {
	return Theme{
		Primary:   lipgloss.NewStyle().Bold(true),
		Secondary: lipgloss.NewStyle(),
		Muted:     lipgloss.NewStyle(),
		Strong:    lipgloss.NewStyle().Bold(true),
		Success:   lipgloss.NewStyle().Bold(true),
		Warning:   lipgloss.NewStyle().Bold(true),
		Error:     lipgloss.NewStyle().Bold(true),
		Info:      lipgloss.NewStyle(),
		Command:   lipgloss.NewStyle().Bold(true),
		Package:   lipgloss.NewStyle().Bold(true),
		Version:   lipgloss.NewStyle(),
		Path:      lipgloss.NewStyle(),
		Code:      lipgloss.NewStyle(),
		Number:    lipgloss.NewStyle(),
		Added:     lipgloss.NewStyle().Bold(true),
		Updated:   lipgloss.NewStyle().Bold(true),
		Removed:   lipgloss.NewStyle().Bold(true),
		Reused:    lipgloss.NewStyle(),
		Header:    lipgloss.NewStyle().Bold(true),
		Label:     lipgloss.NewStyle().Bold(true),
		Value:     lipgloss.NewStyle(),
	}
}

func fg(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
}

func applyStyle(style lipgloss.Style, text string, useColor bool) string {
	if !useColor || text == "" {
		return text
	}
	return style.Render(text)
}
