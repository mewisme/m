package npm

import "github.com/mewisme/mew/internal/apperr"

const mutationUnsupportedMsg = "npm package-lock.json semantic mutation is not supported; use npm to update the lockfile or migrate to m.lock"

func errEmptyRef(subject string) error {
	return apperr.New(apperr.Lockfile, "npm.refresolve", subject, "dangling dependency reference")
}

// ErrMutationUnsupported reports that Mew cannot rewrite an incumbent npm lockfile.
func ErrMutationUnsupported(op, subject string) *apperr.Error {
	return apperr.New(apperr.Unsupported, op, subject, mutationUnsupportedMsg)
}
