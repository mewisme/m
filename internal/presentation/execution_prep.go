package presentation

import (
	"fmt"
	"io"
	"strings"

	"github.com/mewisme/mew/internal/diagnostics"
)

// ExecutionPrepView is the human banner shown before child launch.
type ExecutionPrepView struct {
	Title  string
	Rows   []KeyValue
	Stages []string // cold mx only: Resolving, Consent, Fetching, Preparing
}

// MapEnvironmentPrepared builds a safe prep view from the frozen event + command label.
// Digests are omitted unless debug is true. Absolute paths are never included.
func MapEnvironmentPrepared(ev diagnostics.EnvironmentPreparedEvent, command string, debug bool) ExecutionPrepView {
	title := "Running"
	if command != "" {
		title = "Running " + command
	}
	view := ExecutionPrepView{Title: title}

	sourceLabel, envLabel, networkLabel, integrityLabel := mapPreparedLabels(ev)
	if sourceLabel != "" {
		view.Rows = append(view.Rows, KeyValue{Key: "Source", Value: sourceLabel, Style: ValueMuted})
	}
	if command != "" && sourceLabel != "project" {
		view.Rows = append(view.Rows, KeyValue{Key: "Package", Value: command, Style: ValuePackage})
	}
	if envLabel != "" {
		view.Rows = append(view.Rows, KeyValue{Key: "Environment", Value: envLabel, Style: ValueMuted})
	}
	if networkLabel != "" {
		view.Rows = append(view.Rows, KeyValue{Key: "Network", Value: networkLabel, Style: ValueMuted})
	}
	if integrityLabel != "" {
		view.Rows = append(view.Rows, KeyValue{Key: "Integrity", Value: integrityLabel, Style: ValueMuted})
	}
	if debug {
		if ev.IdentityDigest != "" {
			view.Rows = append(view.Rows, KeyValue{Key: "Identity", Value: shortDigest(ev.IdentityDigest), Style: ValueMuted})
		}
		if ev.GraphDigest != "" {
			view.Rows = append(view.Rows, KeyValue{Key: "Graph", Value: shortDigest(ev.GraphDigest), Style: ValueMuted})
		}
	}
	return view
}

// ProjectExecPrep builds a thin local run/exec banner without inventing EnvironmentPrepared.
func ProjectExecPrep(command, packageName string) ExecutionPrepView {
	title := "Running"
	if command != "" {
		title = "Running " + command
	}
	view := ExecutionPrepView{Title: title}
	view.Rows = append(view.Rows, KeyValue{Key: "Source", Value: "project", Style: ValueMuted})
	pkg := packageName
	if pkg == "" {
		pkg = command
	}
	if pkg != "" {
		view.Rows = append(view.Rows, KeyValue{Key: "Package", Value: pkg, Style: ValuePackage})
	}
	return view
}

func mapPreparedLabels(ev diagnostics.EnvironmentPreparedEvent) (source, env, network, integrity string) {
	switch strings.ToLower(strings.TrimSpace(ev.Source)) {
	case "project":
		source = "project"
	case "dlx":
		source = "dlx"
	case "snapshot":
		source = "snapshot " + shortID(ev.IdentityDigest)
	case "capsule":
		source = "capsule"
	default:
		if ev.Source != "" {
			source = ev.Source
		}
	}

	switch strings.ToLower(strings.TrimSpace(ev.CacheState)) {
	case "warm-hit", "warm":
		env = "warm cache"
	case "project":
		if source == "" {
			source = "project"
		}
	case "ephemeral":
		env = "ephemeral"
	}

	switch strings.ToLower(strings.TrimSpace(ev.Source)) {
	case "snapshot", "capsule":
		// Authoritative LockedProviderPolicy: NetworkForbidden + VerificationRequired.
		if !ev.NetworkUsed {
			network = "disabled"
		}
		env = "verified"
	}
	return source, env, network, ""
}

func shortID(digest string) string {
	d := strings.TrimSpace(digest)
	if len(d) >= 6 {
		return d[:6]
	}
	if d == "" {
		return "unknown"
	}
	return d
}

func shortDigest(digest string) string {
	d := strings.TrimSpace(digest)
	if len(d) > 12 {
		return d[:12] + "…"
	}
	return d
}

// RenderExecutionPrep formats a prep view with arrow title, optional stages, and KV rows.
func RenderExecutionPrep(view ExecutionPrepView, settings EffectiveSettings) string {
	sym := settings.Symbols
	arrow := sym.Arrow
	if arrow == "" {
		arrow = "->"
	}
	var b strings.Builder
	for _, stage := range view.Stages {
		stage = strings.TrimSpace(stage)
		if stage == "" {
			continue
		}
		b.WriteString("  ")
		b.WriteString(stage)
		b.WriteByte('\n')
	}
	b.WriteString(arrow)
	b.WriteByte(' ')
	b.WriteString(strings.TrimSpace(view.Title))
	if len(view.Rows) > 0 {
		b.WriteByte('\n')
		kv := NewStaticRenderer(settings).KeyValues(view.Rows)
		// Indent KV block with two spaces for visual grouping.
		for i, line := range strings.Split(kv, "\n") {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString("  ")
			b.WriteString(line)
		}
	}
	return b.String()
}

// WriteExecutionPrep writes the prep banner to stderr (w).
func WriteExecutionPrep(w io.Writer, view ExecutionPrepView, settings EffectiveSettings) {
	if w == nil {
		return
	}
	text := RenderExecutionPrep(view, settings)
	if text == "" {
		return
	}
	_, _ = fmt.Fprintln(w, text)
}
