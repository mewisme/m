package helpmd

import (
	"strings"

	"charm.land/glamour/v2"
)

// RenderRich renders Markdown with Glamour for rich human TTYs.
func RenderRich(md string, opts RenderOptions) (string, error) {
	md = reHTMLComment.ReplaceAllString(md, "")
	md = reImage.ReplaceAllString(md, "")
	// Strip raw HTML; Glamour should not execute untrusted markup.
	md = reHTML.ReplaceAllString(md, "")

	style := opts.Style
	if style == "" {
		style = "dark"
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath(style),
		glamour.WithWordWrap(opts.Width),
	)
	if err != nil {
		return "", err
	}
	out, err := r.Render(md)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(out, "\n") + "\n", nil
}
