package presentation_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
)

func TestActivityProgressRendererLazyStart(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	// No bytes before first progress event.
	if buf.Len() != 0 {
		t.Fatalf("wrote before progress: %q", buf.String())
	}

	r.Suspend()
	r.Resume()

	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("lazy renderer wrote bytes: %q", buf.String())
	}
}

func TestActivityProgressRendererStartStop(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "install/1/resolve", Kind: "Resolving dependencies",
	})

	// Should have a frame with \r and spinner char.
	out := buf.String()
	if !strings.Contains(out, "\r") {
		t.Fatalf("expected carriage return: %q", out)
	}
	if !strings.Contains(out, "Resolving dependencies") {
		t.Fatalf("expected label: %q", out)
	}

	r.OperationCompleted(diagnostics.OperationCompletedEvent{
		ID: "install/1/resolve", Status: "ok", DurationMs: 10,
	})

	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	// Double Close must be safe.
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestActivityProgressRendererCountsUpdate(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	total := int64(32)
	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "fetch", Kind: "Fetching packages", Total: &total, Unit: "packages",
	})

	// Update progress.
	r.OperationProgress(diagnostics.OperationProgressEvent{
		ID: "fetch", Completed: 24, Total: &total,
	})

	out := buf.String()
	if !strings.Contains(out, "24/32") {
		t.Fatalf("expected counts after progress update: %q", out)
	}

	_ = r.Close()
}

func TestActivityProgressRendererSuspendResume(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	total := int64(10)
	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "op", Kind: "Linking", Total: &total,
	})
	buf.Reset()

	// Suspend clears the line.
	r.Suspend()
	out := buf.String()
	if !strings.Contains(out, "\r") {
		t.Fatalf("suspend should clear line: %q", out)
	}
	buf.Reset()

	// Events while suspended are silently dropped.
	r.OperationProgress(diagnostics.OperationProgressEvent{
		ID: "op", Completed: 5, Total: &total,
	})
	if buf.Len() != 0 {
		t.Fatalf("events while suspended: %q", buf.String())
	}

	// Resume restores the spinner.
	r.Resume()
	out = buf.String()
	if !strings.Contains(out, "Linking") {
		t.Fatalf("resume should redraw: %q", out)
	}

	_ = r.Close()
}

func TestActivityProgressRendererUnicodeFrames(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: true,
		Width:      80,
		Symbols:    presentation.UnicodeSymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "x", Kind: "Test", Label: "Testing",
	})
	out := buf.String()
	// Must contain a Braille spinner char (⣾ or similar).
	found := false
	for _, frame := range []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"} {
		if strings.Contains(out, frame) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Unicode spinner frame: %q", out)
	}
	_ = r.Close()
}

func TestActivityProgressRendererASCIINoColor(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
		UseColor:   false,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "x", Kind: "Resolving",
	})
	out := buf.String()
	// \r and \x1b[K are cursor control, not color — they're needed for the
	// single-line spinner even in no-color mode.
	if strings.Contains(out, "\x1b[38;2;29;78;216m") {
		t.Fatalf("no-color output has color ANSI: %q", out)
	}
	_ = r.Close()
}

func TestActivityProgressRendererColorOutput(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
		UseColor:   true,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "x", Kind: "Resolving", Label: "Resolving",
	})
	out := buf.String()
	if !strings.Contains(out, "\x1b[38;2;29;78;216m") {
		t.Fatalf("color output missing cyan: %q", out)
	}
	_ = r.Close()
}

func TestActivityProgressRendererNoticeDurable(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	// Pre-start notice: durable line only, no spinner.
	r.Notice(diagnostics.NoticeEvent{
		Severity: "warning", Message: "Something is wrong",
	})
	out := buf.String()
	if !strings.Contains(out, "Something is wrong") {
		t.Fatalf("notice not written: %q", out)
	}
	if !strings.Contains(out, "\n") {
		t.Fatalf("notice must end with newline: %q", out)
	}
	_ = r.Close()
}

func TestActivityProgressRendererNoticeMidProgress(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "x", Kind: "Fetching", Label: "Fetching packages",
	})
	buf.Reset()

	r.Notice(diagnostics.NoticeEvent{
		Severity: "warning", Message: "Retry in 2s",
	})
	out := buf.String()
	if !strings.Contains(out, "\r\x1b[K") {
		t.Fatalf("notice should clear line before writing: %q", out)
	}
	if !strings.Contains(out, "Retry in 2s") {
		t.Fatalf("notice missing: %q", out)
	}
	// Spinner should resume after notice.
	if !strings.Contains(out, "Fetching") {
		t.Fatalf("spinner not resumed after notice: %q", out)
	}
	_ = r.Close()
}

func TestActivityProgressRendererMultipleOps(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "op1", Kind: "Resolving", Label: "Resolving",
	})
	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "op2", Kind: "Fetching", Label: "Fetching",
	})

	// Complete op2 (the most recently touched): should switch to op1.
	r.OperationCompleted(diagnostics.OperationCompletedEvent{
		ID: "op2", Status: "ok",
	})
	out := buf.String()
	if !strings.Contains(out, "Resolving") {
		t.Fatalf("after completing current, should show remaining: %q", out)
	}

	// Complete op1: spinner stops.
	r.OperationCompleted(diagnostics.OperationCompletedEvent{
		ID: "op1", Status: "ok",
	})
	// Line should be cleared.
	if !strings.Contains(buf.String(), "\r\x1b[K") {
		t.Fatalf("after last completion, line should be cleared: %q", buf.String())
	}

	_ = r.Close()
}

func TestActivityProgressRendererWidthTruncation(t *testing.T) {
	for _, width := range []int{20, 40, 60, 80, 120} {
		t.Run("", func(t *testing.T) {
			var buf bytes.Buffer
			settings := presentation.EffectiveSettings{
				UseUnicode: false,
				Width:      width,
				Symbols:    presentation.ASCIISymbols,
			}
			r := presentation.NewActivityProgressRenderer(&buf, settings)

			total := int64(999)
			r.OperationStarted(diagnostics.OperationStartedEvent{
				ID: "x", Kind: "Fetching packages from registry.npmjs.org", Total: &total,
			})
			r.OperationProgress(diagnostics.OperationProgressEvent{
				ID: "x", Completed: 500, Total: &total,
			})

			// Build a line from the output and check its display width.
			out := buf.String()
			if strings.Count(out, "\r") == 0 && strings.Count(out, "\x1b[K") == 0 {
				t.Fatalf("width=%d no frame output: %q", width, out)
			}

			// Last line should not exceed width.
			lines := strings.Split(out, "\r")
			last := lines[len(lines)-1]
			// Strip CSI sequences for display width check.
			stripped := stripCSI(last)
			if presentation.CellWidth(stripped) > width+5 {
				t.Fatalf("width=%d last line too wide (%d > %d): %q",
					width, presentation.CellWidth(stripped), width, last)
			}

			_ = r.Close()
		})
	}
}

func TestActivityProgressRendererDiscardWriter(t *testing.T) {
	r := presentation.NewActivityProgressRenderer(io.Discard, presentation.EffectiveSettings{
		Width: 80, Symbols: presentation.ASCIISymbols,
	})
	r.OperationStarted(diagnostics.OperationStartedEvent{ID: "a", Kind: "link", Label: "link"})
	r.Notice(diagnostics.NoticeEvent{Severity: "warning", Message: "blocked"})
	_ = r.Close()
}

func TestActivityProgressRendererConcurrentEvents(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	total := int64(100)
	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "op", Kind: "Processing", Total: &total,
	})

	// Send several concurrent progress events — must not panic.
	done := make(chan struct{})
	go func() {
		for i := int64(0); i < 50; i++ {
			r.OperationProgress(diagnostics.OperationProgressEvent{
				ID: "op", Completed: i, Total: &total,
			})
		}
		close(done)
	}()
	<-done

	_ = r.Close()
}

func TestActivityProgressRendererCloseRacingWithTick(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "x", Kind: "Test", Label: "Testing",
	})

	// Close immediately while ticker is running — must not deadlock.
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestActivityProgressRendererEmptyEvents(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	// Empty kind/label should be ignored.
	r.OperationStarted(diagnostics.OperationStartedEvent{ID: "x"})
	r.OperationCompleted(diagnostics.OperationCompletedEvent{ID: "x"})
	// Empty notice should be ignored.
	r.Notice(diagnostics.NoticeEvent{})

	if buf.Len() != 0 {
		t.Fatalf("wrote on empty events: %q", buf.String())
	}
	_ = r.Close()
}

func TestActivityProgressRendererNoANSILeak(t *testing.T) {
	var buf bytes.Buffer
	settings := presentation.EffectiveSettings{
		UseUnicode: true,
		Width:      80,
		Symbols:    presentation.UnicodeSymbols,
		UseColor:   true,
	}
	r := presentation.NewActivityProgressRenderer(&buf, settings)

	r.OperationStarted(diagnostics.OperationStartedEvent{
		ID: "x", Kind: "Resolving", Label: "Resolving dependencies",
	})
	r.OperationCompleted(diagnostics.OperationCompletedEvent{
		ID: "x", Status: "ok",
	})
	_ = r.Close()

	out := buf.String()
	// Must not contain alt-screen sequences, cursor hide/show, or DEC mode queries.
	for _, bad := range []string{"\x1b[?1049", "\x1b[?25", "\x1b[?2026", "\x1b[?2027"} {
		if strings.Contains(out, bad) {
			t.Fatalf("leaked sequence %q: %q", bad, out)
		}
	}
}

// stripCSI removes ANSI CSI sequences for display-width measurement.
func stripCSI(s string) string {
	for {
		start := strings.Index(s, "\x1b[")
		if start < 0 {
			return s
		}
		end := start + 2
		for end < len(s) && (s[end] < 'A' || s[end] > 'z') {
			end++
		}
		if end < len(s) {
			end++
		}
		s = s[:start] + s[end:]
	}
}
