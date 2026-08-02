package presentation_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/presentation/testkit"
)

func TestPlainProgressRendererPhases(t *testing.T) {
	var buf bytes.Buffer
	p := presentation.NewPlainProgressRenderer(&buf)
	total := int64(42)
	p.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "install/1/resolve", Kind: "resolve", Label: "Resolving",
	})
	p.OperationCompleted(diagnostics.OperationCompletedEvent{
		ID: "install/1/resolve", Status: "ok", DurationMs: 118,
	})
	p.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "install/1/fetch", Kind: "fetch", Total: &total, Unit: "packages",
	})
	p.OperationCompleted(diagnostics.OperationCompletedEvent{
		ID: "install/1/fetch", Status: "ok", DurationMs: 620,
		Metrics: []diagnostics.Metric{
			{Name: "downloaded", Value: 7},
			{Name: "reused", Value: 35},
		},
	})
	p.Notice(diagnostics.NoticeEvent{
		Severity: "warning",
		Message:  "3 lifecycle scripts were blocked",
	})
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	wantLines := []string{
		"resolve started",
		"resolve completed duration=118ms",
		"fetch started packages=42",
		"fetch completed duration=620ms downloaded=7 reused=35",
		"warning lifecycle-blocked count=3",
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line) {
			t.Fatalf("missing %q in:\n%s", line, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("ANSI leaked into plain progress: %q", got)
	}
}

func TestPlainProgressSuspendDropsEvents(t *testing.T) {
	var buf bytes.Buffer
	p := presentation.NewPlainProgressRenderer(&buf)
	p.Suspend()
	p.OperationStarted(diagnostics.OperationStartedEvent{ID: "x", Kind: "fetch"})
	p.Resume()
	p.OperationStarted(diagnostics.OperationStartedEvent{ID: "y", Kind: "link"})
	if got := buf.String(); got != "link started\n" {
		t.Fatalf("got %q", got)
	}
}

func TestPlainProgressCancelledAndFailed(t *testing.T) {
	var buf bytes.Buffer
	p := presentation.NewPlainProgressRenderer(&buf)
	p.OperationStarted(diagnostics.OperationStartedEvent{ID: "a", Kind: "resolve"})
	p.OperationCompleted(diagnostics.OperationCompletedEvent{ID: "a", Status: "cancelled", DurationMs: 5})
	p.OperationStarted(diagnostics.OperationStartedEvent{ID: "b", Kind: "commit"})
	p.OperationCompleted(diagnostics.OperationCompletedEvent{ID: "b", Status: "failed", DurationMs: 2})
	got := buf.String()
	if !strings.Contains(got, "resolve cancelled duration=5ms") {
		t.Fatalf("%q", got)
	}
	if !strings.Contains(got, "commit failed duration=2ms") {
		t.Fatalf("%q", got)
	}
}

func TestControllerAttachesPlainProgress(t *testing.T) {
	// Rich output without TTY attaches static rich progress.
	var errb bytes.Buffer
	resolved, err := presentation.Resolve(presentation.Input{OutputFlag: "rich"})
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Progress {
		t.Fatal("expected progress for rich output")
	}
	ctrl, err := presentation.NewController(resolved, testkit.PipeCapabilities(), presentation.StreamWriters{
		Out: bytes.NewBuffer(nil), Err: &errb,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rep := ctrl.Reporter()
	rep.OperationStarted(diagnostics.OperationStartedEvent{ID: "install/1/resolve", Kind: "resolve"})
	rep.OperationCompleted(diagnostics.OperationCompletedEvent{ID: "install/1/resolve", Status: "ok", DurationMs: 10})
	if !strings.Contains(errb.String(), "resolve") {
		t.Fatalf("stderr=%q", errb.String())
	}
}

func TestControllerProgressNeverSuppressesPhaseLines(t *testing.T) {
	var errb bytes.Buffer
	resolved, err := presentation.Resolve(presentation.Input{
		OutputFlag: "plain", NoProgress: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctrl, err := presentation.NewController(resolved, testkit.PipeCapabilities(), presentation.StreamWriters{
		Out: bytes.NewBuffer(nil), Err: &errb,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctrl.Reporter().OperationStarted(diagnostics.OperationStartedEvent{ID: "x", Kind: "resolve"})
	if errb.Len() != 0 {
		t.Fatalf("--no-progress must suppress phase lines: %q", errb.String())
	}
	ctrl.Reporter().Notice(diagnostics.NoticeEvent{Severity: "warning", Message: "lifecycle blocked"})
	if !strings.Contains(errb.String(), "lifecycle") {
		t.Fatalf("notices must remain visible: %q", errb.String())
	}
}

func TestWritePlainInstallSummary(t *testing.T) {
	var buf bytes.Buffer
	presentation.WritePlainInstallSummary(&buf, 4, 2, 1, 1800)
	got := buf.String()
	if got != "6 packages installed, 1 package removed [1.8s]\n" {
		t.Fatalf("%q", got)
	}
}
