package helpmd_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/presentation"
	helpmd "github.com/mewisme/mew/internal/presentation/help"
)

func TestRenderRichUsesGlamourLayout(t *testing.T) {
	md := "# Errors\n\nUse `m help errors` for codes.\n\n- one\n- two\n"
	out, err := helpmd.RenderRich(md, helpmd.RenderOptions{Width: 72, Style: "dark"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Errors") {
		t.Fatalf("missing heading text:\n%s", out)
	}
	// Glamour list marker (plain renderer uses "-").
	if !strings.Contains(out, "•") {
		t.Fatalf("expected Glamour bullet:\n%s", out)
	}
	if strings.Contains(out, "<script>") {
		t.Fatalf("html leaked:\n%s", out)
	}
}

func TestRenderRichLightStyle(t *testing.T) {
	md := "## Light\n\nbody text\n"
	out, err := helpmd.RenderRich(md, helpmd.RenderOptions{Width: 60, Style: "light"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Light") {
		t.Fatalf("missing heading:\n%s", out)
	}
}

func TestRenderRichStripsHTMLAndImages(t *testing.T) {
	md := "Hello <script>alert(1)</script> ![x](http://evil) world"
	out, err := helpmd.RenderRich(md, helpmd.RenderOptions{Width: 80, Style: "dark"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "script") || strings.Contains(out, "http://evil") {
		t.Fatalf("unsafe content remained:\n%s", out)
	}
}

func TestRenderPlainFlagSkipsGlamour(t *testing.T) {
	md := "# Hello\n\n**bold** and `code`\n\n- item\n"
	out, err := helpmd.Render(md, helpmd.RenderOptions{Width: 80, Plain: true, Style: "dark"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("plain path emitted SGR:\n%q", out)
	}
	if strings.Contains(out, "•") {
		t.Fatalf("plain path used Glamour bullet:\n%s", out)
	}
	if !strings.Contains(out, "Hello") || !strings.Contains(out, "- item") {
		t.Fatalf("missing plain content:\n%s", out)
	}
}

func TestRenderForceColorKeepsANSI(t *testing.T) {
	md := "# Hello\n\n- item\n"
	out, err := helpmd.Render(md, helpmd.RenderOptions{
		Width: 80, Style: "dark", ForceColor: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("ForceColor should keep Glamour SGR:\n%q", out)
	}
	if strings.Contains(out, "# Hello") {
		t.Fatalf("expected Glamour heading, got plain markdown:\n%s", out)
	}
}

func TestGlamourStyleMapsThemeMode(t *testing.T) {
	cases := []struct {
		mode presentation.ThemeMode
		want string
	}{
		{presentation.ThemeLight, "light"},
		{presentation.ThemeDark, "dark"},
		{presentation.ThemeAccessible, "notty"},
		{presentation.ThemeNone, "notty"},
		{"", "dark"},
	}
	for _, tc := range cases {
		if got := helpmd.GlamourStyle(tc.mode); got != tc.want {
			t.Fatalf("GlamourStyle(%q)=%q want %q", tc.mode, got, tc.want)
		}
	}
	if helpmd.GlamourStyle(presentation.ThemeLight) == helpmd.GlamourStyle(presentation.ThemeDark) {
		t.Fatal("light and dark must map to different Glamour styles")
	}
}

func TestRenderRichThemeSelectsStyle(t *testing.T) {
	md := "# Theme\n\n`code` and **bold**\n"
	dark, err := helpmd.RenderRich(md, helpmd.RenderOptions{
		Width: 72, Theme: presentation.ThemeDark, ForceColor: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	light, err := helpmd.RenderRich(md, helpmd.RenderOptions{
		Width: 72, Theme: presentation.ThemeLight, ForceColor: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dark == light {
		t.Fatalf("dark and light Theme must produce different Glamour output")
	}
	if !strings.Contains(dark, "\x1b[") || !strings.Contains(light, "\x1b[") {
		t.Fatalf("expected ANSI in both:\ndark=%q\nlight=%q", dark, light)
	}
}
