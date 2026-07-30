package helpmd

import (
	"strings"

	"charm.land/glamour/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/mewisme/mew/internal/presentation"
)

// GlamourStyle maps effective ThemeMode to a Glamour WithStandardStyle name.
// light→light, dark→dark, accessible/none→notty; empty/auto fall through to dark.
func GlamourStyle(mode presentation.ThemeMode) string {
	switch mode {
	case presentation.ThemeLight:
		return "light"
	case presentation.ThemeAccessible, presentation.ThemeNone:
		return "notty"
	default:
		return "dark"
	}
}

// RenderRich renders Markdown with Glamour for rich human TTYs.
func RenderRich(md string, opts RenderOptions) (string, error) {
	md = reHTMLComment.ReplaceAllString(md, "")
	md = reImage.ReplaceAllString(md, "")
	// Strip raw HTML before render; Glamour has no DisableHTML option and
	// would otherwise sanitize then echo remaining markup text.
	md = reHTML.ReplaceAllString(md, "")

	style := opts.Style
	if style == "" {
		style = GlamourStyle(opts.Theme)
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(opts.Width),
	)
	if err != nil {
		return "", err
	}
	out, err := r.Render(md)
	if err != nil {
		return "", err
	}
	// Glamour is pure and does not downsample. Lip Gloss maps colors to the
	// detected stdout profile — which strips SGR when stdout is non-TTY.
	// Skip that when the caller forced color (--color=always / --output=rich).
	if !opts.ForceColor {
		out = lipgloss.Sprint(out)
	}
	return strings.TrimRight(out, "\n") + "\n", nil
}
