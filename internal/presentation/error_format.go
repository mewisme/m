package presentation

import "strings"

func formatError(view ErrorView, settings EffectiveSettings, color bool, theme Theme) string {
	if view.Title == "" && view.Message == "" {
		return ""
	}
	var parts []string
	title := view.Title
	if !color {
		if title != "" {
			title = "ERROR " + title
		}
	} else {
		title = applyStyle(theme.Error, title, true)
		if sym := statusSymbol(settings.Symbols, StatusError); sym != "" {
			title = applyStyle(theme.Error, sym, true) + " " + title
		}
	}
	if title != "" {
		parts = append(parts, title)
	}
	if view.Message != "" {
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, view.Message)
	}
	if len(view.Context) > 0 {
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, formatKeyValues(view.Context, settings, color, theme))
	}
	if view.Code != "" {
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		codeLine := FormatErrorCode(view.Code)
		if color {
			codeLine = applyStyle(theme.Muted, codeLine, true)
		}
		parts = append(parts, codeLine)
	}
	for _, h := range view.Hints {
		if h.Message == "" {
			continue
		}
		parts = append(parts, formatHintLine(h, settings, color, theme))
	}
	for _, c := range view.Causes {
		if c.Message == "" {
			continue
		}
		line := c.Label + ": " + c.Message
		if color {
			line = applyStyle(theme.Muted, line, true)
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, "\n")
}

func formatHintLine(h Hint, settings EffectiveSettings, color bool, theme Theme) string {
	arrow := settings.Symbols.Arrow
	if arrow == "" {
		return h.Message
	}
	if color {
		arrow = applyStyle(theme.Primary, arrow, true)
	}
	return arrow + " " + h.Message
}
