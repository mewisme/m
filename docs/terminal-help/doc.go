// Package terminalhelp embeds curated terminal-help Markdown for `m help` topics.
//
// These files are short, version-matched terminal topics. Authoritative long-form
// documentation remains under docs/*.md; each topic ends with a See also pointer.
package terminalhelp

import "embed"

// FS holds curated topic Markdown (no network fetch).
//
//go:embed *.md errors/*.md
var FS embed.FS
