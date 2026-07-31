package presentation_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/presentation/testkit"
)

func TestLiveInstallRendererStartStop(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r, err := presentation.NewLiveInstallRenderer(&buf, settings)
	if err != nil {
		t.Fatal(err)
	}
	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "install/1/resolve", Kind: "resolve", Label: "Resolving dependencies",
	})
	time.Sleep(30 * time.Millisecond)
	r.OperationCompleted(diagnostics.OperationCompletedEvent{
		ID: "install/1/resolve", Status: "ok", DurationMs: 10,
	})
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLiveInstallRendererLazyStartNoTerminalIO(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r, err := presentation.NewLiveInstallRenderer(&buf, settings)
	if err != nil {
		t.Fatal(err)
	}
	r.Suspend()
	r.Resume()
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("lazy renderer wrote before progress events: %q", buf.String())
	}
	if strings.Contains(buf.String(), "2027") {
		t.Fatalf("mode 2027 query leaked: %q", buf.String())
	}
}

func TestControllerPlainOutputUsesPlainProgress(t *testing.T) {
	// Plain output no longer has progress by default. Verify no progress sink is created.
	resolved, err := presentation.Resolve(presentation.Input{OutputFlag: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Output != presentation.OutputPlain {
		t.Fatalf("output=%s", resolved.Output)
	}
	if resolved.Progress {
		t.Fatal("plain output should not have progress")
	}
}

func TestControllerRichTTYUsesLiveProgress(t *testing.T) {
	caps := testkit.TTYCapabilities()
	resolved, err := presentation.Resolve(presentation.Input{OutputFlag: "rich"})
	if err != nil {
		t.Fatal(err)
	}
	settings := presentation.Effective(resolved, caps)
	if !settings.UseProgress {
		t.Fatal("expected UseProgress on TTY rich")
	}

	// With non-TTY caps, rich should use static-rich progress.
	pipe := testkit.PipeCapabilities()
	resolved2, err := presentation.Resolve(presentation.Input{OutputFlag: "rich"})
	if err != nil {
		t.Fatal(err)
	}
	var errb bytes.Buffer
	ctrl, err := presentation.NewController(resolved2, pipe, presentation.StreamWriters{
		Out: bytes.NewBuffer(nil), Err: &errb,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctrl.Reporter().OperationStarted(diagnostics.OperationStartedEvent{ID: "x", Kind: "validate"})
	if !strings.Contains(errb.String(), "validate") {
		t.Fatalf("expected static-rich progress: %q", errb.String())
	}
}
