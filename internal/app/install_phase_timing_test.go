package app

import (
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/diagnostics"
)

type phaseDebugReporter struct {
	lines []string
}

func (r *phaseDebugReporter) Progress(diagnostics.Event) {}
func (r *phaseDebugReporter) Error(error)                {}
func (r *phaseDebugReporter) Debug(msg string, attrs ...diagnostics.Attr) {
	if msg != "install phase" {
		return
	}
	var phase, elapsed string
	for _, a := range attrs {
		switch a.Key {
		case "phase":
			phase = a.Value
		case "elapsed":
			elapsed = a.Value
		}
	}
	if phase != "" && elapsed != "" {
		r.lines = append(r.lines, phase+" "+elapsed)
	}
}

func (r *phaseDebugReporter) WorkspaceTask(diagnostics.WorkspaceTaskEvent) {}
func (r *phaseDebugReporter) ChildOutput(diagnostics.ChildOutputEvent, diagnostics.WorkspaceOutputMode) {
}
func (r *phaseDebugReporter) WorkspaceSummary(diagnostics.WorkspaceSummaryEvent) {}
func (r *phaseDebugReporter) EnvironmentPrepared(diagnostics.EnvironmentPreparedEvent) error {
	return nil
}
func (r *phaseDebugReporter) OperationStarted(diagnostics.OperationStartedEvent)     {}
func (r *phaseDebugReporter) OperationProgress(diagnostics.OperationProgressEvent)   {}
func (r *phaseDebugReporter) OperationCompleted(diagnostics.OperationCompletedEvent) {}
func (r *phaseDebugReporter) Notice(diagnostics.NoticeEvent)                         {}

func TestInstallPhaseDebugTiming(t *testing.T) {
	rep := &phaseDebugReporter{}
	ac := &Context{Reporter: rep}
	timer := startInstallPhase(ac, "fetch")
	time.Sleep(2 * time.Millisecond)
	timer.done()

	if len(rep.lines) != 1 {
		t.Fatalf("lines=%v", rep.lines)
	}
	if !strings.HasPrefix(rep.lines[0], "fetch ") {
		t.Fatalf("got %q", rep.lines[0])
	}
	if !strings.Contains(rep.lines[0], "ms") && !strings.Contains(rep.lines[0], "s") {
		t.Fatalf("missing elapsed duration: %q", rep.lines[0])
	}
}

func TestInstallPhaseDebugSilentWithoutReporter(t *testing.T) {
	timer := startInstallPhase(nil, "resolve")
	timer.done()
	timer = startInstallPhase(&Context{}, "resolve")
	timer.done()
}
