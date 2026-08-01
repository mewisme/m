package presentation_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
)

type errWriter struct {
	n int
}

func (w *errWriter) Write(p []byte) (int, error) {
	w.n++
	if w.n > 1 {
		return 0, errors.New("broken pipe")
	}
	return len(p), nil
}

func TestPlainProgressBrokenPipeDoesNotPanic(t *testing.T) {
	w := &errWriter{}
	p := presentation.NewPlainProgressRenderer(w)
	p.OperationStarted(diagnostics.OperationStartedEvent{ID: "a", Kind: "resolve"})
	p.OperationCompleted(diagnostics.OperationCompletedEvent{ID: "a", Status: "ok", DurationMs: 1})
	p.OperationStarted(diagnostics.OperationStartedEvent{ID: "b", Kind: "fetch"})
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPlainProgressDiscardWriter(t *testing.T) {
	p := presentation.NewPlainProgressRenderer(io.Discard)
	p.OperationStarted(diagnostics.OperationStartedEvent{ID: "a", Kind: "link"})
	p.Notice(diagnostics.NoticeEvent{Severity: "warning", Message: "2 lifecycle scripts were blocked"})
	_ = p.Close()
}

func TestActivityProgressRendererBrokenPipeClose(t *testing.T) {
	w := &errWriter{}
	settings := presentation.EffectiveSettings{Width: 80, Symbols: presentation.ASCIISymbols}
	r := presentation.NewActivityProgressRenderer(w, settings)
	r.OperationStarted(diagnostics.OperationStartedEvent{ID: "a", Kind: "resolve", Label: "resolve"})
	// Close must return even after write errors.
	_ = r.Close()
	_ = r.Close()
}

func TestMutationSummaryHelpersCompile(t *testing.T) {
	// Ensure package still builds when only presentation tests run.
	var buf bytes.Buffer
	presentation.WritePlainInstallSummary(&buf, 0, 0, 0, 0)
	if buf.String() != "installed added=0 updated=0 removed=0\n" {
		t.Fatalf("%q", buf.String())
	}
}
