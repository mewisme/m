package presentation_test

import (
	"testing"

	"github.com/mewisme/mew/internal/presentation"
)

func TestParseMarkdownTheme(t *testing.T) {
	cases := []struct {
		raw     string
		want    presentation.MarkdownTheme
		wantErr bool
	}{
		{"", presentation.MarkdownThemeDark, false},
		{"dark", presentation.MarkdownThemeDark, false},
		{"DARK", presentation.MarkdownThemeDark, false},
		{"  dark  ", presentation.MarkdownThemeDark, false},
		{"light", presentation.MarkdownThemeLight, false},
		{"dracula", presentation.MarkdownThemeDracula, false},
		{"tokyo-night", presentation.MarkdownThemeTokyoNight, false},
		{"notty", presentation.MarkdownThemeNoTTY, false},
		{"unknown", "", true},
		{"  unknown  ", "", true},
		{"/etc/passwd", "", true},
		{"https://evil.com", "", true},
	}
	for _, tc := range cases {
		got, err := presentation.ParseMarkdownTheme(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseMarkdownTheme(%q): expected error, got %q", tc.raw, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseMarkdownTheme(%q): unexpected error: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ParseMarkdownTheme(%q)=%q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestParseMarkdownThemeWhitespaceOnly(t *testing.T) {
	got, err := presentation.ParseMarkdownTheme("   ")
	if err != nil {
		t.Fatalf("whitespace-only should default: %v", err)
	}
	if got != presentation.MarkdownThemeDark {
		t.Fatalf("whitespace-only: got %q, want dark", got)
	}
}

func TestResolveMarkdownTheme(t *testing.T) {
	cases := []struct {
		configured presentation.MarkdownTheme
		noColor    bool
		ascii      bool
		want       presentation.MarkdownTheme
	}{
		{"", false, false, presentation.MarkdownThemeDark},
		{presentation.MarkdownThemeDark, false, false, presentation.MarkdownThemeDark},
		{presentation.MarkdownThemeLight, false, false, presentation.MarkdownThemeLight},
		{presentation.MarkdownThemeDracula, false, false, presentation.MarkdownThemeDracula},
		{presentation.MarkdownThemeTokyoNight, false, false, presentation.MarkdownThemeTokyoNight},
		{presentation.MarkdownThemeNoTTY, false, false, presentation.MarkdownThemeNoTTY},
		// noColor overrides to notty
		{presentation.MarkdownThemeDark, true, false, presentation.MarkdownThemeNoTTY},
		{presentation.MarkdownThemeLight, true, false, presentation.MarkdownThemeNoTTY},
		{presentation.MarkdownThemeDracula, true, false, presentation.MarkdownThemeNoTTY},
		// ascii overrides to ascii (internal)
		{presentation.MarkdownThemeDark, false, true, "ascii"},
		{presentation.MarkdownThemeLight, false, true, "ascii"},
		// ascii takes precedence over noColor
		{presentation.MarkdownThemeDark, true, true, "ascii"},
	}
	for _, tc := range cases {
		got := presentation.ResolveMarkdownTheme(tc.configured, tc.noColor, tc.ascii)
		if got != tc.want {
			t.Fatalf("ResolveMarkdownTheme(%q, noColor=%v, ascii=%v)=%q, want %q",
				tc.configured, tc.noColor, tc.ascii, got, tc.want)
		}
	}
}

func TestSupportedMarkdownThemes(t *testing.T) {
	themes := presentation.SupportedMarkdownThemes()
	if len(themes) != 5 {
		t.Fatalf("expected 5 themes, got %d: %v", len(themes), themes)
	}
	want := []presentation.MarkdownTheme{
		presentation.MarkdownThemeDark,
		presentation.MarkdownThemeLight,
		presentation.MarkdownThemeDracula,
		presentation.MarkdownThemeTokyoNight,
		presentation.MarkdownThemeNoTTY,
	}
	for i, th := range themes {
		if th != want[i] {
			t.Fatalf("SupportedMarkdownThemes()[%d]=%q, want %q", i, th, want[i])
		}
	}
}

func TestMarkdownThemeGlamourStyle(t *testing.T) {
	cases := []struct {
		theme presentation.MarkdownTheme
		want  string
	}{
		{presentation.MarkdownThemeDark, "dark"},
		{presentation.MarkdownThemeLight, "light"},
		{presentation.MarkdownThemeDracula, "dracula"},
		{presentation.MarkdownThemeTokyoNight, "tokyo-night"},
		{presentation.MarkdownThemeNoTTY, "notty"},
		{presentation.MarkdownTheme(""), "dark"},
		{presentation.MarkdownTheme("unknown"), "dark"},
	}
	for _, tc := range cases {
		got := tc.theme.GlamourStyle()
		if got != tc.want {
			t.Fatalf("MarkdownTheme(%q).GlamourStyle()=%q, want %q", tc.theme, got, tc.want)
		}
	}
}

func TestDefaultMarkdownTheme(t *testing.T) {
	if presentation.DefaultMarkdownTheme != presentation.MarkdownThemeDark {
		t.Fatalf("DefaultMarkdownTheme=%q, want dark", presentation.DefaultMarkdownTheme)
	}
}
