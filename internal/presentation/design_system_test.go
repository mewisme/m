package presentation_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/presentation"
)

func TestNoColorDisablesColor(t *testing.T) {
	got, err := presentation.Resolve(presentation.Input{NoColor: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Color {
		t.Fatalf("color should be false with --no-color")
	}
}

func TestDefaultColorEnabledForRich(t *testing.T) {
	got, err := presentation.Resolve(presentation.Input{})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Color {
		t.Fatal("color should default to true for rich output")
	}
}

func TestDetectCapabilitiesInjected(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "TERM":
			return "dumb", true
		case "CI":
			return "1", true
		case "COLORFGBG":
			return "15;0", true
		default:
			return "", false
		}
	}
	caps := presentation.DetectCapabilities(nil, bytes.NewBuffer(nil), bytes.NewBuffer(nil), lookup)
	if !caps.DumbTerminal || !caps.CI {
		t.Fatalf("%+v", caps)
	}
	if caps.Background != presentation.BackgroundDark {
		t.Fatalf("background=%q", caps.Background)
	}
	if caps.Width != 80 {
		t.Fatalf("width=%d", caps.Width)
	}
}

func TestPlainRendererNoANSI(t *testing.T) {
	settings := presentation.EffectiveSettings{
		UseColor: false, UseUnicode: false, ThemeMode: presentation.ThemeNone,
		Width: 80, Symbols: presentation.ASCIISymbols,
	}
	r := presentation.NewStaticRenderer(settings)
	out := r.Summary(presentation.Summary{
		Status: presentation.StatusSuccess,
		Title:  "Installed 3 packages",
		Metrics: []presentation.KeyValue{
			{Key: "Packages", Value: "3", Style: presentation.ValueNumber},
		},
		Notices: []presentation.Notice{{Status: presentation.StatusWarning, Message: "scripts blocked"}},
		Hints:   []presentation.Hint{{Message: "Run `m builds`"}},
	})
	if strings.ContainsAny(out, "\x1b") {
		t.Fatalf("ANSI in plain output: %q", out)
	}
	if !strings.Contains(out, "OK Installed") {
		t.Fatalf("%q", out)
	}
	goldenPath := filepath.Join("testdata", "summary-ascii-80.txt")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if normalize(out) != normalize(string(want)) {
		t.Fatalf("golden mismatch\ngot:\n%s\nwant:\n%s", out, want)
	}
}

func TestUnicodeSymbolsAndWidths(t *testing.T) {
	settings := presentation.EffectiveSettings{
		UseColor: false, UseUnicode: true, ThemeMode: presentation.ThemeNone,
		Width: 80, Symbols: presentation.UnicodeSymbols,
	}
	r := presentation.NewStaticRenderer(settings)
	out := r.Status(presentation.StatusLine{
		Status: presentation.StatusSuccess,
		Text:   "Added zod",
		Detail: "4.0.14",
	})
	if !strings.HasPrefix(out, "\u2713 ") {
		t.Fatalf("%q", out)
	}
	if bad := presentation.ValidateSymbolWidths(presentation.UnicodeSymbols); len(bad) != 0 {
		t.Fatalf("unicode symbol widths: %v", bad)
	}
	if bad := presentation.ValidateSymbolWidths(presentation.ASCIISymbols); len(bad) != 0 {
		t.Fatalf("ascii symbol widths: %v", bad)
	}
	deltas := r.PackageDeltas([]presentation.PackageDelta{
		{Kind: presentation.DeltaAdded, Name: "zod", Version: "4.0.14"},
		{Kind: presentation.DeltaUpdated, Name: "react", From: "19.1.0", To: "19.1.1"},
		{Kind: presentation.DeltaRemoved, Name: "lodash", Version: "4.17.21"},
	})
	if strings.ContainsAny(deltas, "\x1b") {
		t.Fatalf("ANSI: %q", deltas)
	}
}

func TestTableWidthsAndStacked(t *testing.T) {
	model := presentation.TableModel{
		Columns: []presentation.TableColumn{
			{Key: "pkg", Header: "PACKAGE", MinWidth: 4, Prefer: 20, Primary: true},
			{Key: "cur", Header: "CURRENT", MinWidth: 4, Prefer: 10},
			{Key: "want", Header: "WANTED", MinWidth: 4, Prefer: 10},
		},
		Rows: []map[string]string{
			{"pkg": "react", "cur": "19.1.0", "want": "19.1.1"},
			{"pkg": "typescript", "cur": "5.8.2", "want": "5.8.3"},
		},
	}
	for _, width := range []int{40, 60, 80, 120} {
		settings := presentation.EffectiveSettings{
			ThemeMode: presentation.ThemeNone, Width: width,
			Symbols: presentation.ASCIISymbols,
		}
		r := presentation.NewStaticRenderer(settings)
		out := r.Table(model)
		if strings.ContainsAny(out, "\x1b") {
			t.Fatalf("width %d ANSI: %q", width, out)
		}
		name := filepath.Join("testdata", "table-"+strconv.Itoa(width)+".txt")
		want, err := os.ReadFile(name)
		if os.IsNotExist(err) {
			t.Fatalf("missing golden %s; got:\n%s", name, out)
		}
		if err != nil {
			t.Fatal(err)
		}
		if normalize(out) != normalize(string(want)) {
			t.Fatalf("width %d mismatch\ngot:\n%s\nwant:\n%s", width, out, want)
		}
	}
}

func TestRichThemeEmitsANSI(t *testing.T) {
	settings := presentation.EffectiveSettings{
		UseColor: true, UseUnicode: true, ThemeMode: presentation.ThemeDark,
		Width: 80, Symbols: presentation.UnicodeSymbols,
	}
	r := presentation.NewStaticRenderer(settings)
	out := r.Status(presentation.StatusLine{Status: presentation.StatusSuccess, Text: "ok"})
	if !strings.Contains(out, "\x1b") {
		t.Fatalf("expected ANSI styling: %q", out)
	}
}

func TestOutputModeGatesStaticColor(t *testing.T) {
	richCaps := presentation.Capabilities{
		StdoutTTY: true, StderrTTY: true,
		ColorProfile: presentation.ColorProfileTrueColor,
		Width:        80, Unicode: true,
	}
	status := presentation.StatusLine{Status: presentation.StatusSuccess, Text: "ok"}

	t.Run("tty rich emits ANSI", func(t *testing.T) {
		opts, err := presentation.Resolve(presentation.Input{})
		if err != nil {
			t.Fatal(err)
		}
		if opts.Output != presentation.OutputRich {
			t.Fatalf("output=%s", opts.Output)
		}
		eff := presentation.Effective(opts, richCaps, nil)
		if !eff.UseColor {
			t.Fatal("expected UseColor")
		}
		out := presentation.NewStaticRenderer(eff).Status(status)
		if !strings.Contains(out, "\x1b") {
			t.Fatalf("expected ANSI: %q", out)
		}
	})

	t.Run("explicit plain no ANSI on color TTY", func(t *testing.T) {
		opts, err := presentation.Resolve(presentation.Input{OutputFlag: "plain"})
		if err != nil {
			t.Fatal(err)
		}
		if opts.Output != presentation.OutputPlain {
			t.Fatalf("output=%s", opts.Output)
		}
		eff := presentation.Effective(opts, richCaps, nil)
		if eff.UseColor {
			t.Fatal("expected no UseColor for --output=plain")
		}
		out := presentation.NewStaticRenderer(eff).Status(status)
		if strings.ContainsAny(out, "\x1b") {
			t.Fatalf("ANSI in plain mode: %q", out)
		}
	})

	t.Run("no color no ANSI", func(t *testing.T) {
		opts, err := presentation.Resolve(presentation.Input{NoColor: true})
		if err != nil {
			t.Fatal(err)
		}
		eff := presentation.Effective(opts, richCaps, nil)
		if eff.UseColor {
			t.Fatal("expected no UseColor")
		}
		out := presentation.NewStaticRenderer(eff).Status(status)
		if strings.ContainsAny(out, "\x1b") {
			t.Fatalf("ANSI with --no-color: %q", out)
		}
	})

	t.Run("plain wins over no color double", func(t *testing.T) {
		opts, err := presentation.Resolve(presentation.Input{
			OutputFlag: "plain",
			NoColor:    true,
		})
		if err != nil {
			t.Fatal(err)
		}
		eff := presentation.Effective(opts, richCaps, nil)
		if eff.UseColor {
			t.Fatal("--output=plain must suppress ANSI")
		}
	})
}

func TestMiddleTruncateAndCJK(t *testing.T) {
	got := presentation.MiddleTruncate("abcdefghijklmnopqrstuvwxyz", 10, "...")
	if presentation.CellWidth(got) > 10 {
		t.Fatalf("%q width=%d", got, presentation.CellWidth(got))
	}
	path := `C:\Users\Mew\very\long\path\to\package.json`
	got = presentation.MiddleTruncate(path, 24, "...")
	if presentation.CellWidth(got) > 24 {
		t.Fatalf("%q", got)
	}
	cjk := presentation.CellWidth("\u65e5\u672c\u8a9e\u30d1\u30c3\u30b1\u30fc\u30b8")
	if cjk < 8 {
		t.Fatalf("cjk width=%d", cjk)
	}
}

func TestOutputModeDirectMapping(t *testing.T) {
	cases := []struct {
		name string
		in   presentation.Input
		want presentation.OutputMode
	}{
		{"default rich", presentation.Input{}, presentation.OutputRich},
		{"explicit rich", presentation.Input{OutputFlag: "rich"}, presentation.OutputRich},
		{"explicit plain", presentation.Input{OutputFlag: "plain"}, presentation.OutputPlain},
		{"explicit json", presentation.Input{OutputFlag: "json"}, presentation.OutputJSON},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := presentation.Resolve(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.Output != tc.want {
				t.Fatalf("got %q want %q", got.Output, tc.want)
			}
		})
	}
}

func TestKeyValuesStackedNarrow(t *testing.T) {
	settings := presentation.EffectiveSettings{
		ThemeMode: presentation.ThemeNone, Width: 40, Symbols: presentation.ASCIISymbols,
	}
	r := presentation.NewStaticRenderer(settings)
	out := r.KeyValues([]presentation.KeyValue{
		{Key: "Packages", Value: "126"},
		{Key: "Downloaded", Value: "7"},
	})
	if !strings.Contains(out, "Packages: 126") {
		t.Fatalf("%q", out)
	}
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimRight(s, "\n") + "\n"
}
