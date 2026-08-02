package presentation_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
)

func TestControllerSuspendStopsWorkspaceStatus(t *testing.T) {
	var errBuf bytes.Buffer
	caps := presentation.Capabilities{StderrTTY: true, Width: 80}
	resolved := presentation.ResolvedOptions{
		Output:    presentation.OutputPlain,
		Summary:   true,
		Progress:  false,
		TermWidth: 80,
	}
	ctrl, err := presentation.NewController(resolved, caps, presentation.StreamWriters{
		Out: bytes.NewBuffer(nil),
		Err: &errBuf,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rep := ctrl.Reporter()
	rep.WorkspaceTask(diagnostics.WorkspaceTaskEvent{Package: "a", Script: "t", Status: "start", Index: 0})
	if errBuf.Len() == 0 {
		t.Fatal("expected status line before suspend")
	}
	errBuf.Reset()
	if err := ctrl.Suspend(context.Background()); err != nil {
		t.Fatal(err)
	}
	rep.WorkspaceTask(diagnostics.WorkspaceTaskEvent{Package: "a", Script: "t", Status: "done", Index: 0})
	if errBuf.Len() != 0 {
		t.Fatalf("status must not print while suspended: %q", errBuf.String())
	}
	if err := ctrl.Resume(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestHumanModesSkipStructuredPrep(t *testing.T) {
	var out bytes.Buffer
	rep := diagnostics.NewReporter(diagnostics.Options{
		Format: "ndjson",
		Out:    &out,
		Err:    bytes.NewBuffer(nil),
	})
	rep.Progress(diagnostics.Event{Type: "prep-stage", Phase: "Resolving x"})
	rep.Progress(diagnostics.Event{Type: "exec-prep", Phase: "Running x"})
	rep.Progress(diagnostics.Event{Type: "exec-summary", Phase: "x"})
	if out.Len() != 0 {
		t.Fatalf("ndjson must ignore human-only prep events: %q", out.String())
	}
}
