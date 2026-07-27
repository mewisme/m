package lifecycle

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/policy"
)

// CheckTrust decides whether lifecycle scripts for pkg may run.
func CheckTrust(pkg string, eff *config.Effective, trusted *TrustStore, interactive bool, stdin io.Reader, stdout io.Writer) error {
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
		if !interactive {
			return apperr.New(apperr.Policy, "lifecycle.trust", pkg,
				"package not trusted and interactive approval disabled")
		}
		if stdin == nil {
			stdin = os.Stdin
		}
		if stdout == nil {
			stdout = os.Stdout
		}
		_, _ = fmt.Fprintf(stdout, "Trust lifecycle scripts for %s? [y/N] ", pkg)
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && err != io.EOF {
			return apperr.Wrap(apperr.IO, "lifecycle.trust", pkg, err)
		}
		if strings.EqualFold(strings.TrimSpace(line), "y") {
			if trusted != nil {
				if err := trusted.AddTrusted(pkg); err != nil {
					return err
				}
			}
			return nil
		}
		return apperr.New(apperr.Policy, "lifecycle.trust", pkg, "package not trusted")
	default:
		return apperr.New(apperr.Config, "lifecycle.trust", pkg, "unknown lifecycle.script_trust")
	}
}
