package npm

import (
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

func errEmptyRef(subject string) error {
	return apperr.New(apperr.Lockfile, "npm.refresolve", subject, "dangling dependency reference")
}

// ErrMutationUnsupported reports that Mew cannot rewrite an incumbent npm lockfile.
func ErrMutationUnsupported(op, subject string) *apperr.Error {
	return apperr.New(apperr.Unsupported, op, subject, mutationUnsupportedMessage(subject))
}

func mutationUnsupportedMessage(subject string) string {
	lock := npmLockLabel(subject)
	return fmt.Sprintf(`cannot rewrite %s while the project keeps npm identity
incumbent npm locks are read-only; only byte-preserving no-ops are allowed

to update dependencies:
  npm install
  m install

to migrate to m.lock:
  m lock migrate --from npm --to m --dry-run
  m lock migrate --from npm --to m`, lock)
}

func npmLockLabel(subject string) string {
	switch strings.ToLower(strings.TrimSpace(subject)) {
	case "package-lock.json":
		return "package-lock.json"
	case "npm-shrinkwrap.json":
		return "npm-shrinkwrap.json"
	default:
		if subject == "" || subject == "." {
			return "package-lock.json"
		}
		return subject
	}
}
