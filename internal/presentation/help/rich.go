package helpmd

import (
	"strings"

	"charm.land/glamour/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// RenderRich renders Markdown with Glamour for rich human TTYs.
func RenderRich(md string, opts RenderOptions) (string, error) {
	md = reHTMLComment.ReplaceAllString(md, "")
	md = reImage.ReplaceAllString(md, "")
	// Strip raw HTML before render; Glamour has no DisableHTML option and
	// would otherwise sanitize then echo remaining markup text.
	md = reHTML.ReplaceAllString(md, "")

	style := opts.Style
	if style == "" {
		style = "dark"
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
	// Glamour is pure and does not downsample; Lip Gloss maps colors to the
	// terminal profile (same role as lipgloss.Print in the Glamour README).
	out = lipgloss.Sprint(out)
	return strings.TrimRight(out, "\n") + "\n", nil
}
