package dlx_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/prompt"
	"github.com/mewisme/mew/internal/runner/dlx"
)

func TestEvaluateConsentYesSkipsPrompt(t *testing.T) {
	d := dlx.EvaluateConsent(false, dlx.ConsentStore{}, dlx.ConsentKey{}, true, false)
	if !d.Approved {
		t.Fatal("expected approved")
	}
}

func TestEvaluateConsentNonInteractiveNeedTTY(t *testing.T) {
	d := dlx.EvaluateConsent(false, dlx.ConsentStore{}, dlx.ConsentKey{}, false, false)
	if !d.NeedTTY {
		t.Fatal("expected NeedTTY")
	}
}

func TestPromptConsentApprove(t *testing.T) {
	p := &prompt.ScriptedPrompter{Answers: []prompt.PromptAnswer{{OptionID: prompt.OptionApprove}}}
	ok, err := dlx.PromptConsent(context.Background(), p, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected approve")
	}
}

func TestPromptConsentDenyDefault(t *testing.T) {
	p := &prompt.ScriptedPrompter{Answers: []prompt.PromptAnswer{{OptionID: prompt.OptionReject}}}
	ok, err := dlx.PromptConsent(context.Background(), p, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected deny")
	}
}

func TestPromptConsentNilPrompter(t *testing.T) {
	ok, err := dlx.PromptConsent(context.Background(), nil, "abc123")
	if err == nil || ok {
		t.Fatal("expected usage error")
	}
}
