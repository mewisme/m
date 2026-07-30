package prompt_test

import (
	"testing"

	"github.com/mewisme/mew/internal/prompt"
)

func TestResolveInteractiveAuto(t *testing.T) {
	caps := prompt.Caps{
		StdinTTY: true, HumanMode: true, AccessibleOK: true, RichOK: true,
	}
	d := prompt.ResolveInteractive(prompt.InteractiveAuto, caps)
	if !d.CanPrompt {
		t.Fatalf("expected CanPrompt, got %+v", d)
	}
}

func TestResolveInteractiveAutoCI(t *testing.T) {
	caps := prompt.Caps{
		StdinTTY: true, HumanMode: true, CI: true, AccessibleOK: true, RichOK: true,
	}
	d := prompt.ResolveInteractive(prompt.InteractiveAuto, caps)
	if d.CanPrompt || !d.DeniedByPolicy {
		t.Fatalf("expected CI deny, got %+v", d)
	}
}

func TestResolveInteractiveAutoJSON(t *testing.T) {
	caps := prompt.Caps{
		StdinTTY: true, HumanMode: false, AccessibleOK: true, RichOK: true,
	}
	d := prompt.ResolveInteractive(prompt.InteractiveAuto, caps)
	if d.CanPrompt {
		t.Fatalf("structured mode must not prompt: %+v", d)
	}
}

func TestResolveInteractiveAlwaysNeedsTTY(t *testing.T) {
	caps := prompt.Caps{StdinTTY: false, HumanMode: true, AccessibleOK: true, RichOK: true}
	d := prompt.ResolveInteractive(prompt.InteractiveAlways, caps)
	if d.CanPrompt || !d.NeedTTY {
		t.Fatalf("always without TTY must NeedTTY: %+v", d)
	}
}

func TestResolveInteractiveNever(t *testing.T) {
	caps := prompt.Caps{StdinTTY: true, HumanMode: true, AccessibleOK: true, RichOK: true}
	d := prompt.ResolveInteractive(prompt.InteractiveNever, caps)
	if d.CanPrompt || !d.DeniedByPolicy {
		t.Fatalf("never must deny: %+v", d)
	}
}

func TestResolveInteractiveAccessibleFallback(t *testing.T) {
	caps := prompt.Caps{
		StdinTTY: true, HumanMode: true, Accessible: true, AccessibleOK: true, RichOK: true,
	}
	d := prompt.ResolveInteractive(prompt.InteractiveAuto, caps)
	if !d.CanPrompt || !d.UseAccessible {
		t.Fatalf("accessible mode should use numbered adapter: %+v", d)
	}
}

func TestSafeDefaultID(t *testing.T) {
	if got := prompt.SafeDefaultID(prompt.PromptRequest{Dangerous: true, DefaultID: prompt.OptionDeny}); got != prompt.OptionDeny {
		t.Fatalf("deny default: %q", got)
	}
	if got := prompt.SafeDefaultID(prompt.PromptRequest{Dangerous: true, DefaultID: prompt.OptionTrustProject}); got != "" {
		t.Fatalf("dangerous non-deny default must be empty on EOF, got %q", got)
	}
	if got := prompt.SafeDefaultID(prompt.PromptRequest{Dangerous: false, DefaultID: "x"}); got != "x" {
		t.Fatalf("safe non-dangerous: %q", got)
	}
}

func TestValidateRequestDuplicateOptions(t *testing.T) {
	err := prompt.ValidateRequest(prompt.PromptRequest{
		Kind: prompt.PromptSelect,
		Options: []prompt.Option{
			{ID: "a", Label: "A"},
			{ID: "a", Label: "B"},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate option error")
	}
}
