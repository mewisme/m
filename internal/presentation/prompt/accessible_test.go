package prompt_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	presprompt "github.com/mewisme/mew/internal/presentation/prompt"
	mewprompt "github.com/mewisme/mew/internal/prompt"
)

func trustReq() mewprompt.PromptRequest {
	return mewprompt.PromptRequest{
		ID:        "lifecycle.trust",
		Kind:      mewprompt.PromptSelect,
		Title:     "Package esbuild requests permission to run an install script.",
		Dangerous: true,
		DefaultID: mewprompt.OptionDeny,
		Fields: []mewprompt.Field{
			{Key: "Package", Value: "esbuild"},
			{Key: "Script", Value: "install"},
		},
		Options: []mewprompt.Option{
			{ID: mewprompt.OptionDeny, Label: "Deny"},
			{ID: mewprompt.OptionAllowOnce, Label: "Allow once"},
			{ID: mewprompt.OptionTrustProject, Label: "Trust for this project"},
		},
	}
}

func TestAccessibleSelectNumber(t *testing.T) {
	var errW bytes.Buffer
	p := presprompt.NewAccessible(presprompt.Options{
		Stdin:  strings.NewReader("2\n"),
		Stderr: &errW,
		Width:  40,
	})
	ans, err := p.Prompt(context.Background(), trustReq())
	if err != nil {
		t.Fatal(err)
	}
	if ans.OptionID != mewprompt.OptionAllowOnce {
		t.Fatalf("got %q", ans.OptionID)
	}
	out := errW.String()
	if !strings.Contains(out, "1. Deny") || !strings.Contains(out, "Package: esbuild") {
		t.Fatalf("unexpected output:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatal("accessible output must be ANSI-free")
	}
}

func TestAccessibleEOFDenies(t *testing.T) {
	p := presprompt.NewAccessible(presprompt.Options{
		Stdin:  strings.NewReader(""),
		Stderr: ioDiscard{},
		Width:  40,
	})
	ans, err := p.Prompt(context.Background(), trustReq())
	if err != nil {
		t.Fatal(err)
	}
	if ans.OptionID != mewprompt.OptionDeny {
		t.Fatalf("EOF must deny, got %+v", ans)
	}
}

func TestAccessibleEmptyEnterDenies(t *testing.T) {
	p := presprompt.NewAccessible(presprompt.Options{
		Stdin:  strings.NewReader("\n"),
		Stderr: ioDiscard{},
		Width:  40,
	})
	ans, err := p.Prompt(context.Background(), trustReq())
	if err != nil {
		t.Fatal(err)
	}
	if ans.OptionID != mewprompt.OptionDeny {
		t.Fatalf("empty enter must deny, got %+v", ans)
	}
}

func TestAccessibleInvalidThenSelect(t *testing.T) {
	p := presprompt.NewAccessible(presprompt.Options{
		Stdin:  strings.NewReader("9\n1\n"),
		Stderr: ioDiscard{},
		Width:  40,
	})
	ans, err := p.Prompt(context.Background(), trustReq())
	if err != nil {
		t.Fatal(err)
	}
	if ans.OptionID != mewprompt.OptionDeny {
		t.Fatalf("got %q", ans.OptionID)
	}
}

func TestAccessibleSecretNotEchoedInValue(t *testing.T) {
	var errW bytes.Buffer
	p := presprompt.NewAccessible(presprompt.Options{
		Stdin:  strings.NewReader("super-secret\n"),
		Stderr: &errW,
		Width:  40,
	})
	ans, err := p.Prompt(context.Background(), mewprompt.PromptRequest{
		Kind:   mewprompt.PromptInput,
		Title:  "Token",
		Secret: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ans.Value != "" {
		t.Fatalf("secret value must be cleared, got %q", ans.Value)
	}
	if strings.Contains(errW.String(), "super-secret") {
		t.Fatal("secret must not appear in stderr snapshot")
	}
}

func TestSuspendCalledBeforePrompt(t *testing.T) {
	var order []string
	p := presprompt.New(presprompt.Options{
		Accessible: true,
		Stdin:      strings.NewReader("1\n"),
		Stderr:     ioDiscard{},
		Suspend: func(context.Context) error {
			order = append(order, "suspend")
			return nil
		},
		Resume: func(context.Context) error {
			order = append(order, "resume")
			return nil
		},
	})
	_, err := p.Prompt(context.Background(), trustReq())
	if err != nil {
		t.Fatal(err)
	}
	if len(order) < 2 || order[0] != "suspend" || order[len(order)-1] != "resume" {
		t.Fatalf("order=%v", order)
	}
}

func TestFactoryPrefersAccessible(t *testing.T) {
	p := presprompt.New(presprompt.Options{
		Accessible: true,
		UseRich:    true,
		Stdin:      strings.NewReader("1\n"),
		Stderr:     ioDiscard{},
	})
	ans, err := p.Prompt(context.Background(), trustReq())
	if err != nil {
		t.Fatal(err)
	}
	if ans.OptionID != mewprompt.OptionDeny {
		t.Fatalf("got %q", ans.OptionID)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
