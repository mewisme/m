package presentation

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/diagnostics"
)

// CompletionSummary is the optional post-child stderr line.
type CompletionSummary struct {
	Name      string
	Duration  time.Duration
	ExitCode  int
	Failed    bool
	Cancelled bool
}

// ShouldEmitCompletionSummary reports whether a wrapper summary is allowed.
func ShouldEmitCompletionSummary(opts ResolvedOptions, intent TerminalIntent, interactiveChild bool) bool {
	if !opts.Summary {
		return false
	}
	if opts.Structured() || opts.Output == OutputSilent {
		return false
	}
	if interactiveChild || intent == TerminalInteractive {
		return false
	}
	return true
}

// RenderCompletionSummary formats "✓ name completed in 1.2s" (or failed/cancelled).
func RenderCompletionSummary(s CompletionSummary, settings EffectiveSettings) string {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = "command"
	}
	sym := settings.Symbols
	status := StatusSuccess
	verb := "completed"
	prefix := sym.Success
	if s.Cancelled {
		status = StatusWarning
		verb = "cancelled"
		prefix = sym.Warning
	} else if s.Failed || s.ExitCode != 0 {
		status = StatusError
		verb = "failed"
		prefix = sym.Error
	}
	_ = status
	var b strings.Builder
	if prefix != "" {
		b.WriteString(prefix)
		b.WriteByte(' ')
	}
	b.WriteString(name)
	b.WriteByte(' ')
	b.WriteString(verb)
	if s.Duration > 0 {
		b.WriteString(" in ")
		b.WriteString(FormatDuration(s.Duration))
	}
	return b.String()
}

// WriteCompletionSummary writes the summary to stderr, inserting a leading newline
// when ensureNL is true (child output lacked a final newline).
func WriteCompletionSummary(w io.Writer, ensureNL bool, s CompletionSummary, settings EffectiveSettings) {
	if w == nil {
		return
	}
	line := RenderCompletionSummary(s, settings)
	if line == "" {
		return
	}
	if ensureNL {
		_, _ = io.WriteString(w, "\n")
	}
	_, _ = fmt.Fprintln(w, line)
}

// AttrRowsToKeyValues maps diagnostics attrs to presentation rows.
func AttrRowsToKeyValues(rows []diagnostics.Attr) []KeyValue {
	out := make([]KeyValue, 0, len(rows))
	for _, a := range rows {
		if strings.TrimSpace(a.Key) == "" {
			continue
		}
		out = append(out, KeyValue{Key: a.Key, Value: a.Value})
	}
	return out
}
