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

// brightANSI returns a lipgloss style with the given ANSI 4-bit bright color.
// These resolve to the terminal's configured bright palette — vivid on dark,
// light, and transparent backgrounds.
func brightANSI(code string) lipgloss.Style {
	return fg(code)
}

func lightTheme() Theme {
	return Theme{
		Primary:   brightANSI("14"),            // bright cyan
		Secondary: brightANSI("8"),             // bright black (gray)
		Muted:     brightANSI("8"),             // bright black (gray)
		Strong:    brightANSI("15").Bold(true), // bright white bold
		Success:   brightANSI("10"),            // bright green
		Warning:   brightANSI("11"),            // bright yellow
		Error:     brightANSI("9"),             // bright red
		Info:      brightANSI("14"),            // bright cyan
		Command:   brightANSI("14"),            // bright cyan
		Package:   brightANSI("14"),            // bright cyan
		Version:   brightANSI("15"),            // bright white
		Path:      brightANSI("8"),             // gray
		Code:      brightANSI("13"),            // bright magenta
		Number:    brightANSI("15"),            // bright white
		Added:     brightANSI("10"),            // bright green
		Updated:   brightANSI("11"),            // bright yellow
		Removed:   brightANSI("9"),             // bright red
		Reused:    brightANSI("8"),             // gray
		Header:    brightANSI("15").Bold(true), // bright white bold
		Label:     brightANSI("15").Bold(true), // bright white bold
		Value:     brightANSI("15"),            // bright white
	}
}

func darkTheme() Theme {
	return Theme{
		Primary:   brightANSI("14"),            // bright cyan
		Secondary: brightANSI("8"),             // bright black (gray)
		Muted:     brightANSI("8"),             // bright black (gray)
		Strong:    brightANSI("15").Bold(true), // bright white bold
		Success:   brightANSI("10"),            // bright green
		Warning:   brightANSI("11"),            // bright yellow
		Error:     brightANSI("9"),             // bright red
		Info:      brightANSI("14"),            // bright cyan
		Command:   brightANSI("14"),            // bright cyan
		Package:   brightANSI("14"),            // bright cyan
		Version:   brightANSI("15"),            // bright white
		Path:      brightANSI("8"),             // gray
		Code:      brightANSI("13"),            // bright magenta
		Number:    brightANSI("15"),            // bright white
		Added:     brightANSI("10"),            // bright green
		Updated:   brightANSI("11"),            // bright yellow
		Removed:   brightANSI("9"),             // bright red
		Reused:    brightANSI("8"),             // gray
		Header:    brightANSI("15").Bold(true), // bright white bold
		Label:     brightANSI("15").Bold(true), // bright white bold
		Value:     brightANSI("15"),            // bright white
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
