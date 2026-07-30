package presentation

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

func TestEveryCodeHasTitle(t *testing.T) {
	for _, code := range apperr.AllCodes() {
		title := TitleForCode(code)
		if title == "" {
			t.Fatalf("missing title for %s", code)
		}
		if code != apperr.OK && title == "Operation failed" {
			t.Fatalf("generic title for %s", code)
		}
	}
}

func TestMapErrorRedactsRegistryURL(t *testing.T) {
	err := apperr.New(apperr.Network, "registry.fetch", "lodash",
		"GET https://user:secret@registry.example.com/lodash failed")
	view := MapError(err, MapOptions{Redact: diagnostics.Redact})
	if strings.Contains(view.Message, "secret") {
		t.Fatalf("message not redacted: %q", view.Message)
	}
	if strings.Contains(view.Subject, "secret") {
		t.Fatalf("subject not redacted: %q", view.Subject)
	}
}

func TestErrorPlainGoldenWidths(t *testing.T) {
	err := apperr.New(apperr.Lockfile, "install", "m.lock", "lockfile does not match package.json")
	for _, width := range []int{40, 80, 120} {
		settings := EffectiveSettings{
			ThemeMode:  ThemeNone,
			Width:      width,
			UseUnicode: false,
			Symbols:    ASCIISymbols,
		}
		got := NewStaticRenderer(settings).Error(MapError(err, MapOptions{Redact: diagnostics.Redact}))
		if !strings.Contains(got, "ERROR Lockfile validation failed") {
			t.Fatalf("width=%d missing title:\n%s", width, got)
		}
		if !strings.Contains(got, "ERR_M_LOCKFILE") {
			t.Fatalf("width=%d missing code:\n%s", width, got)
		}
		if !strings.Contains(got, "m install") {
			t.Fatalf("width=%d missing hint:\n%s", width, got)
		}
	}
}

func TestErrorHintsCapped(t *testing.T) {
	meta := ErrorMetadata{Code: apperr.Usage}
	hints := HintsFor(meta, false)
	if len(hints) > maxDefaultHints {
		t.Fatalf("got %d hints", len(hints))
	}
}

func TestCancelledHasNoHints(t *testing.T) {
	hints := HintsFor(ErrorMetadata{Code: apperr.Cancelled}, false)
	if len(hints) != 0 {
		t.Fatalf("hints=%v", hints)
	}
}

func TestErrorDebugCauses(t *testing.T) {
	inner := apperr.New(apperr.IO, "read", "m.lock", "permission denied")
	err := apperr.Wrap(apperr.Lockfile, "install", "m.lock", inner)
	view := MapError(err, MapOptions{Debug: true, Redact: diagnostics.Redact})
	if len(view.Causes) == 0 {
		t.Fatal("expected debug causes")
	}
}

func TestHumanErrorRenderHook(t *testing.T) {
	settings := EffectiveSettings{ThemeMode: ThemeNone, Width: 80, Symbols: ASCIISymbols}
	render := func(err error) string {
		return NewStaticRenderer(settings).Error(MapError(err, MapOptions{Redact: diagnostics.Redact}))
	}
	got := render(apperr.New(apperr.Usage, "cli", "", "bad flag"))
	if !strings.HasPrefix(got, "ERROR Invalid command usage") {
		t.Fatalf("%q", got)
	}
}
