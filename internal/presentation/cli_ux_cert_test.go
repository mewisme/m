package presentation_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/presentation/testkit"
)

func TestCLIUXWidthsErrorTableSummary(t *testing.T) {
	err := apperr.New(apperr.Lockfile, "install", "m.lock", "lockfile does not match package.json")
	table := presentation.TableModel{
		Columns: []presentation.TableColumn{
			{Key: "pkg", Header: "PACKAGE", MinWidth: 4, Prefer: 20, Primary: true},
			{Key: "cur", Header: "CURRENT", MinWidth: 4, Prefer: 10},
		},
		Rows: []map[string]string{{"pkg": "react", "cur": "19.1.0"}},
	}
	summary := presentation.Summary{
		Status: presentation.StatusSuccess,
		Title:  "Installed 1 package",
		Metrics: []presentation.KeyValue{
			{Key: "Packages", Value: "1", Style: presentation.ValueNumber},
		},
	}
	for _, width := range []int{40, 60, 80, 120} {
		settings := presentation.EffectiveSettings{
			ThemeMode:  presentation.ThemeNone,
			Width:      width,
			UseUnicode: false,
			Symbols:    presentation.ASCIISymbols,
		}
		r := presentation.NewStaticRenderer(settings)
		for _, label := range []string{"error", "table", "summary"} {
			var out string
			switch label {
			case "error":
				out = r.Error(presentation.MapError(err, presentation.MapOptions{Redact: diagnostics.Redact}))
			case "table":
				out = r.Table(table)
			case "summary":
				out = r.Summary(summary)
			}
			if presentation.ContainsCSI([]byte(out)) {
				t.Fatalf("width=%d %s contains CSI: %q", width, label, out)
			}
			if presentation.ContainsCursorControl([]byte(out)) {
				t.Fatalf("width=%d %s contains cursor control: %q", width, label, out)
			}
			if strings.TrimSpace(out) == "" {
				t.Fatalf("width=%d %s empty", width, label)
			}
		}
	}
}

func TestCLIUXAccessibleNoANSI(t *testing.T) {
	resolved, err := presentation.Resolve(presentation.Input{Accessible: true})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Output != presentation.OutputRich {
		t.Fatalf("accessible output=%s want rich", resolved.Output)
	}
	caps := testkit.PipeCapabilities()
	settings := presentation.Effective(resolved, caps, nil)
	if settings.UseColor {
		t.Fatal("accessible must not use color")
	}
	r := presentation.NewStaticRenderer(settings)
	out := r.Summary(presentation.Summary{
		Status: presentation.StatusSuccess,
		Title:  "Done",
	})
	if presentation.ContainsCSI([]byte(out)) {
		t.Fatalf("accessible summary has CSI: %q", out)
	}
	errOut := r.Error(presentation.MapError(
		apperr.New(apperr.Usage, "cli", "", "bad flag"),
		presentation.MapOptions{Redact: diagnostics.Redact},
	))
	if presentation.ContainsCSI([]byte(errOut)) || presentation.ContainsCursorControl([]byte(errOut)) {
		t.Fatalf("accessible error has CSI/cursor: %q", errOut)
	}
	for _, width := range []int{40, 60, 80, 120} {
		settings.Width = width
		rw := presentation.NewStaticRenderer(settings)
		promptish := rw.Status(presentation.StatusLine{Text: "1) allow  2) deny"})
		if presentation.ContainsCSI([]byte(promptish)) {
			t.Fatalf("accessible width=%d has CSI: %q", width, promptish)
		}
	}
}

func TestCLIUXProgressNotOnStdout(t *testing.T) {
	var out, errb bytes.Buffer
	// Rich output without TTY uses static rich progress on stderr.
	resolved, err := presentation.Resolve(presentation.Input{OutputFlag: "rich"})
	if err != nil {
		t.Fatal(err)
	}
	ctrl, err := presentation.NewController(resolved, testkit.PipeCapabilities(), presentation.StreamWriters{
		Out: &out, Err: &errb,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctrl.Reporter().OperationStarted(diagnostics.OperationStartedEvent{ID: "install/1/resolve", Kind: "resolve"})
	ctrl.Reporter().Notice(diagnostics.NoticeEvent{Severity: "warning", Message: "scripts blocked"})
	if out.Len() != 0 {
		t.Fatalf("progress/notices leaked to stdout: %q", out.String())
	}
	if !strings.Contains(errb.String(), "resolve") {
		t.Fatalf("expected progress on stderr: %q", errb.String())
	}
}

func TestCLIUXControllerCleanup(t *testing.T) {
	var errb bytes.Buffer
	resolved, err := presentation.Resolve(presentation.Input{OutputFlag: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	ctrl, err := presentation.NewController(resolved, testkit.PipeCapabilities(), presentation.StreamWriters{
		Out: bytes.NewBuffer(nil), Err: &errb,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ctrl.Reporter().OperationStarted(diagnostics.OperationStartedEvent{ID: "x", Kind: "fetch"})
	if err := ctrl.Suspend(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Close(ctx, presentation.Outcome{}); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Close(ctx, presentation.Outcome{}); err != nil {
		t.Fatal(err)
	}
	if presentation.ContainsCursorControl(errb.Bytes()) {
		t.Fatalf("cleanup left cursor control: %q", errb.String())
	}
}

func TestCLIUXPerformanceAdvisory(t *testing.T) {
	settings := presentation.EffectiveSettings{
		ThemeMode: presentation.ThemeNone, Width: 80, Symbols: presentation.ASCIISymbols,
	}
	r := presentation.NewStaticRenderer(settings)
	start := time.Now()
	const n = 200
	for i := 0; i < n; i++ {
		_ = r.Summary(presentation.Summary{
			Status: presentation.StatusSuccess,
			Title:  "Installed packages",
			Metrics: []presentation.KeyValue{
				{Key: "Packages", Value: "3"},
			},
		})
	}
	elapsed := time.Since(start)
	per := elapsed / n
	if per > 5*time.Millisecond {
		t.Fatalf("advisory: static summary median-ish %s > 5ms (total %s for %d)", per, elapsed, n)
	}
}
