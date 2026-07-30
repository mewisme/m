package lifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/prompt"
)

// CheckTrust decides whether lifecycle scripts for pkg may run.
// When script_trust=ask and Interactive is true, Prompter selects among
// deny / allow-once / trust-project. AllowOnce is request-scoped only.
func CheckTrust(ctx context.Context, pkg string, eff *config.Effective, trusted *TrustStore, interactive bool, prompter prompt.Prompter, allowOnce map[string]struct{}) error {
	mode := config.String(eff, "lifecycle.script_trust", "deny")
	switch policy.ScriptTrust(mode) {
	case policy.ScriptTrustAllow:
		return nil
	case policy.ScriptTrustDeny:
		if trusted != nil && trusted.IsTrusted(pkg) {
			return nil
		}
		return apperr.New(apperr.Policy, "lifecycle.trust", pkg,
			"package not trusted; run m trust "+pkg+" or m approve-builds "+pkg)
	case policy.ScriptTrustAsk:
		if trusted != nil && trusted.IsTrusted(pkg) {
			return nil
		}
		if allowOnce != nil {
			if _, ok := allowOnce[pkg]; ok {
				return nil
			}
		}
		if !interactive || prompter == nil {
			return apperr.New(apperr.Policy, "lifecycle.trust", pkg,
				"package not trusted and interactive approval is unavailable")
		}
		ans, err := prompter.Prompt(ctx, prompt.PromptRequest{
			ID:          "lifecycle.trust",
			Kind:        prompt.PromptSelect,
			Title:       fmt.Sprintf("Package %s requests permission to run an install script.", pkg),
			Description: "Allow this package?",
			Dangerous:   true,
			DefaultID:   prompt.OptionDeny,
			Fields: []prompt.Field{
				{Key: "Package", Value: pkg},
			},
			Options: []prompt.Option{
				{ID: prompt.OptionDeny, Label: "Deny"},
				{ID: prompt.OptionAllowOnce, Label: "Allow once"},
				{ID: prompt.OptionTrustProject, Label: "Trust for this project"},
			},
		})
		if err != nil {
			if errors.Is(err, prompt.ErrCancelled) || ans.Cancelled {
				return apperr.New(apperr.Policy, "lifecycle.trust", pkg, "package not trusted")
			}
			return err
		}
		if ans.Cancelled {
			return apperr.New(apperr.Policy, "lifecycle.trust", pkg, "package not trusted")
		}
		switch ans.OptionID {
		case prompt.OptionTrustProject:
			if trusted != nil {
				if err := trusted.AddTrusted(pkg); err != nil {
					return err
				}
			}
			return nil
		case prompt.OptionAllowOnce:
			if allowOnce != nil {
				allowOnce[pkg] = struct{}{}
			}
			return nil
		default:
			return apperr.New(apperr.Policy, "lifecycle.trust", pkg, "package not trusted")
		}
	default:
		return apperr.New(apperr.Config, "lifecycle.trust", pkg, "unknown lifecycle.script_trust")
	}
}
