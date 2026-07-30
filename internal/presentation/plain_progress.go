package presentation

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mewisme/mew/internal/diagnostics"
)

// PlainProgressRenderer writes append-only, zero-ANSI phase lines to stderr.
type PlainProgressRenderer struct {
	mu        sync.Mutex
	out       io.Writer
	suspended bool
	closed    bool
	active    map[string]string // id -> kind
}

// NewPlainProgressRenderer builds an append-only progress sink.
func NewPlainProgressRenderer(w io.Writer) *PlainProgressRenderer {
	if w == nil {
		w = io.Discard
	}
	return &PlainProgressRenderer{out: w, active: map[string]string{}}
}

func (p *PlainProgressRenderer) OperationStarted(ev diagnostics.OperationStartedEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.suspended {
		return
	}
	kind := strings.TrimSpace(ev.Kind)
	if kind == "" {
		kind = strings.TrimSpace(ev.Label)
	}
	if kind == "" {
		return
	}
	p.active[ev.ID] = kind
	line := kind + " started"
	if ev.Total != nil && *ev.Total > 0 {
		unit := ev.Unit
		if unit == "" {
			unit = "packages"
		}
		line += fmt.Sprintf(" %s=%d", unit, *ev.Total)
	}
	p.writeln(line)
}

func (p *PlainProgressRenderer) OperationProgress(diagnostics.OperationProgressEvent) {}

func (p *PlainProgressRenderer) OperationCompleted(ev diagnostics.OperationCompletedEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.suspended {
		return
	}
	kind := p.active[ev.ID]
	delete(p.active, ev.ID)
	if kind == "" {
		kind = phaseKindFromID(ev.ID)
	}
	if kind == "" {
		return
	}
	status := strings.TrimSpace(ev.Status)
	if status == "" {
		status = "ok"
	}
	var b strings.Builder
	switch status {
	case "ok":
		b.WriteString(kind)
		b.WriteString(" completed")
	case "skipped":
		b.WriteString(kind)
		b.WriteString(" skipped")
	case "cancelled":
		b.WriteString(kind)
		b.WriteString(" cancelled")
	default:
		b.WriteString(kind)
		b.WriteString(" failed")
	}
	if ev.DurationMs > 0 {
		b.WriteString(" duration=")
		b.WriteString(formatDurationMs(ev.DurationMs))
	}
	metrics := append([]diagnostics.Metric(nil), ev.Metrics...)
	sort.Slice(metrics, func(i, j int) bool { return metrics[i].Name < metrics[j].Name })
	for _, m := range metrics {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(name)
		b.WriteByte('=')
		if m.Value == float64(int64(m.Value)) {
			fmt.Fprintf(&b, "%d", int64(m.Value))
		} else {
			fmt.Fprintf(&b, "%g", m.Value)
		}
	}
	p.writeln(b.String())
}

func (p *PlainProgressRenderer) Notice(ev diagnostics.NoticeEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	sev := strings.TrimSpace(ev.Severity)
	if sev == "" {
		sev = "warning"
	}
	msg := strings.TrimSpace(ev.Message)
	if msg == "" {
		return
	}
	line := sev + " " + sanitizeNoticeToken(msg)
	if strings.Contains(strings.ToLower(msg), "lifecycle") && strings.Contains(strings.ToLower(msg), "blocked") {
		line = "warning lifecycle-blocked"
		if n := extractCount(msg); n >= 0 {
			line += fmt.Sprintf(" count=%d", n)
		}
	}
	p.writeln(line)
}

func (p *PlainProgressRenderer) Suspend() {
	p.mu.Lock()
	p.suspended = true
	p.mu.Unlock()
}

func (p *PlainProgressRenderer) Resume() {
	p.mu.Lock()
	p.suspended = false
	p.mu.Unlock()
}

func (p *PlainProgressRenderer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *PlainProgressRenderer) writeln(line string) {
	_, _ = fmt.Fprintln(p.out, line)
}

func phaseKindFromID(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func formatDurationMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	sec := float64(ms) / 1000
	if sec < 10 {
		return fmt.Sprintf("%.1fs", sec)
	}
	return fmt.Sprintf("%.0fs", sec)
}

func sanitizeNoticeToken(msg string) string {
	msg = strings.ReplaceAll(msg, " ", "-")
	return strings.ToLower(msg)
}

func extractCount(msg string) int {
	for _, f := range strings.Fields(msg) {
		n, err := strconv.Atoi(f)
		if err == nil {
			return n
		}
	}
	return -1
}

// WritePlainInstallSummary emits the final installed= key=value line used in CI logs.
func WritePlainInstallSummary(w io.Writer, added, updated, removed int, durationMs int64) {
	if w == nil {
		return
	}
	line := fmt.Sprintf("installed added=%d updated=%d removed=%d", added, updated, removed)
	if durationMs > 0 {
		line += " duration=" + formatDurationMs(durationMs)
	}
	_, _ = fmt.Fprintln(w, line)
}
