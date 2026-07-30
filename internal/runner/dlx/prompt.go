package dlx

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// PromptConsent asks the user to approve fetch and execute on stderr.
func PromptConsent(w io.Writer, r io.Reader, envDigest string) (bool, error) {
	if w == nil {
		w = io.Discard
	}
	_, _ = fmt.Fprintf(w, "Fetch and run %s?\nInstall scripts remain blocked by lifecycle policy. [y/N] ", envDigest)
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, apperr.Wrap(apperr.IO, "dlx.prompt", "", err)
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes", nil
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
