package dlx

import (
	"context"
	"errors"
	"io"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/prompt"
)

// PromptConsent asks the user to approve fetch and execute via Prompter.
// Prompt text goes to the Prompter's stderr writer; answers come from stdin.
func PromptConsent(ctx context.Context, p prompt.Prompter, envDigest string) (bool, error) {
	if p == nil {
		return false, apperr.New(apperr.Usage, "dlx.prompt", "", "interactive consent requires a TTY")
	}
	safe := diagnostics.Redact(envDigest)
	ans, err := p.Prompt(ctx, prompt.PromptRequest{
		ID:          "dlx.consent",
		Kind:        prompt.PromptConfirm,
		Title:       "Fetch and run this package?",
		Description: "Install scripts remain blocked by lifecycle policy.",
		Dangerous:   true,
		DefaultID:   prompt.OptionReject,
		Fields: []prompt.Field{
			{Key: "Environment", Value: safe},
		},
		Options: []prompt.Option{
			{ID: prompt.OptionReject, Label: "No"},
			{ID: prompt.OptionApprove, Label: "Yes"},
		},
	})
	if err != nil {
		if errors.Is(err, prompt.ErrCancelled) || errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	if ans.Cancelled {
		return false, nil
	}
	return ans.OptionID == prompt.OptionApprove, nil
}

// ConsentDecision captures warm/cold consent matrix outcome.
type ConsentDecision struct {
	Approved bool
	Denied   bool
	NeedTTY  bool
}

// EvaluateConsent applies the warm/cold consent matrix.
func EvaluateConsent(warm bool, prior ConsentStore, key ConsentKey, yes bool, interactive bool) ConsentDecision {
	if prior.HasConsent(key) || yes {
		return ConsentDecision{Approved: true}
	}
	if !interactive {
		return ConsentDecision{NeedTTY: true}
	}
	return ConsentDecision{}
}

// DeniedPolicyError returns ERR_M_POLICY for denied consent.
func DeniedPolicyError() error {
	return apperr.New(apperr.Policy, "dlx.consent", "", "fetch consent denied")
}

// NonInteractiveUsageError returns ERR_M_USAGE for non-TTY without --yes.
func NonInteractiveUsageError() error {
	return apperr.New(apperr.Usage, "dlx.consent", "", "non-interactive mx requires --yes for remote fetch")
}
