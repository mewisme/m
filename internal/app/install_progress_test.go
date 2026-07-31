package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/diagnostics"
)

type opCaptureReporter struct {
	started   []diagnostics.OperationStartedEvent
	progress  []diagnostics.OperationProgressEvent
	completed []diagnostics.OperationCompletedEvent
	notices   []diagnostics.NoticeEvent
	debug     []string
	progressE []diagnostics.Event
}

func (r *opCaptureReporter) Progress(ev diagnostics.Event) {
	r.progressE = append(r.progressE, ev)
}
func (r *opCaptureReporter) Error(error) {}
func (r *opCaptureReporter) Debug(msg string, attrs ...diagnostics.Attr) {
	if msg == "install phase" {
		var phase, status string
		for _, a := range attrs {
			switch a.Key {
			case "phase":
				phase = a.Value
			case "status":
				status = a.Value
			}
		}
		r.debug = append(r.debug, phase+" "+status)
	}
}
func (r *opCaptureReporter) WorkspaceTask(diagnostics.WorkspaceTaskEvent) {}
func (r *opCaptureReporter) ChildOutput(diagnostics.ChildOutputEvent, diagnostics.WorkspaceOutputMode) {
}
func (r *opCaptureReporter) WorkspaceSummary(diagnostics.WorkspaceSummaryEvent) {}
func (r *opCaptureReporter) EnvironmentPrepared(diagnostics.EnvironmentPreparedEvent) error {
	return nil
}
func (r *opCaptureReporter) OperationStarted(ev diagnostics.OperationStartedEvent) {
	r.started = append(r.started, ev)
}
func (r *opCaptureReporter) OperationProgress(ev diagnostics.OperationProgressEvent) {
	r.progress = append(r.progress, ev)
}
func (r *opCaptureReporter) OperationCompleted(ev diagnostics.OperationCompletedEvent) {
	r.completed = append(r.completed, ev)
}
func (r *opCaptureReporter) Notice(ev diagnostics.NoticeEvent) {
	r.notices = append(r.notices, ev)
}

func TestInstallProgressOrderAndStatus(t *testing.T) {
	rep := &opCaptureReporter{}
	ac := &Context{Reporter: rep}
	opID := "1"

	ph := beginInstallPhase(ac, opID, phaseResolve)
	time.Sleep(2 * time.Millisecond)
	ph.Complete(statusOK)

	total := int64(3)
	ph2 := beginInstallPhaseLabeled(ac, opID, phaseFetch, phaseFetch, &total, "packages")
	ph2.Progress(1, "")
	ph2.Progress(2, "")
	ph2.Complete(statusOK,
		diagnostics.Metric{Name: "downloaded", Value: 1, Unit: "packages"},
	)

	ph3 := beginInstallPhase(ac, opID, phaseCommit)
	ph3.CompleteErr(context.Canceled)

	if len(rep.started) != 3 {
		t.Fatalf("started=%d want 3", len(rep.started))
	}
	if rep.started[0].Kind != phaseResolve || rep.started[1].Kind != phaseFetch {
		t.Fatalf("kinds=%v %v", rep.started[0].Kind, rep.started[1].Kind)
	}
	if !strings.HasPrefix(rep.started[0].ID, "install/1/") {
		t.Fatalf("id=%q", rep.started[0].ID)
	}
	if len(rep.progress) != 2 {
		t.Fatalf("progress=%d", len(rep.progress))
	}
	if len(rep.completed) != 3 {
		t.Fatalf("completed=%d", len(rep.completed))
	}
	if rep.completed[0].Status != statusOK || rep.completed[2].Status != statusCancelled {
		t.Fatalf("statuses=%q %q", rep.completed[0].Status, rep.completed[2].Status)
	}
	if rep.completed[0].DurationMs < 0 {
		t.Fatal("duration missing")
	}
	if len(rep.completed[1].Metrics) != 1 || rep.completed[1].Metrics[0].Name != "downloaded" {
		t.Fatalf("metrics=%v", rep.completed[1].Metrics)
	}
}

func TestInstallProgressIgnoresDuplicateComplete(t *testing.T) {
	rep := &opCaptureReporter{}
	ac := &Context{Reporter: rep}
	ph := beginInstallPhase(ac, "9", phaseLink)
	ph.Complete(statusOK)
	ph.Complete(statusFailed)
	ph.CompleteErr(errors.New("nope"))
	if len(rep.completed) != 1 {
		t.Fatalf("completed=%d", len(rep.completed))
	}
	if rep.completed[0].Status != statusOK {
		t.Fatalf("status=%q", rep.completed[0].Status)
	}
}

func TestInstallProgressIgnoresProgressAfterComplete(t *testing.T) {
	rep := &opCaptureReporter{}
	ac := &Context{Reporter: rep}
	total := int64(2)
	ph := beginInstallPhaseLabeled(ac, "2", phaseFetch, phaseFetch, &total, "packages")
	ph.Complete(statusOK)
	ph.Progress(1, "late")
	if len(rep.progress) != 0 {
		t.Fatalf("progress after complete: %v", rep.progress)
	}
}

func TestInstallProgressNoInferFromNextPhase(t *testing.T) {
	rep := &opCaptureReporter{}
	ac := &Context{Reporter: rep}
	ph1 := beginInstallPhase(ac, "3", phaseResolve)
	_ = beginInstallPhase(ac, "3", phaseFetch)
	if len(rep.completed) != 0 {
		t.Fatal("starting next phase must not complete prior phase")
	}
	ph1.Complete(statusOK)
	if len(rep.completed) != 1 || rep.completed[0].ID != "install/3/resolve" {
		t.Fatalf("completed=%v", rep.completed)
	}
}

func TestInstallProgressSilentWithoutReporter(t *testing.T) {
	ph := beginInstallPhase(nil, "1", phaseResolve)
	ph.Complete(statusOK)
	ph = beginInstallPhase(&Context{}, "1", phaseResolve)
	ph.CompleteErr(errors.New("x"))
}

func TestInstallPhaseDebugTiming(t *testing.T) {
	rep := &opCaptureReporter{}
	ac := &Context{Reporter: rep}
	ph := beginInstallPhase(ac, "7", phaseFetch)
	time.Sleep(2 * time.Millisecond)
	ph.Complete(statusOK)
	if len(rep.debug) != 1 || !strings.HasPrefix(rep.debug[0], "fetch ok") {
		t.Fatalf("debug=%v", rep.debug)
	}
}
