package npm

import "github.com/mewisme/mew/internal/apperr"

func errEmptyRef(subject string) error {
	return apperr.New(apperr.Lockfile, "npm.refresolve", subject, "dangling dependency reference")
}
