package helpmd_test

import (
	"strings"
	"testing"

	helpmd "github.com/mewisme/mew/internal/presentation/help"
)

func TestRenderPlainWidthAndLinks(t *testing.T) {
	md := "# Title\n\nSee [docs](docs/cli.md) and `m help runner`.\n\n- one\n- two\n"
	out := helpmd.RenderPlain(md, helpmd.RenderOptions{Width: 40, Plain: true})
	if strings.Contains(out, "\x1b") {
		t.Fatalf("plain output has ANSI:\n%s", out)
	}
	if !strings.Contains(out, "docs: docs/cli.md") && !strings.Contains(out, "docs/cli.md") {
		t.Fatalf("missing link destination:\n%s", out)
	}
}

func TestRenderPlainStripsHTMLAndImages(t *testing.T) {
	md := "Hello <script>alert(1)</script> ![x](http://x) world"
	out := helpmd.RenderPlain(md, helpmd.RenderOptions{Width: 80, Plain: true})
	if strings.Contains(out, "script") || strings.Contains(out, "http://x") {
		t.Fatalf("unsafe content remained:\n%s", out)
	}
}

func TestRenderAccessibleForcesPlain(t *testing.T) {
	md := "# Hello\n\n**bold** text"
	out, err := helpmd.Render(md, helpmd.RenderOptions{Width: 80, Accessible: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("accessible path emitted SGR:\n%q", out)
	}
}
