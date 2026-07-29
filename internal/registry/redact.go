package registry

import (
	"errors"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

func redactErr(err error) error {
	if err == nil {
		return nil
	}
	return apperr.New(apperr.CodeOf(err), errOp(err), errSubject(err), diagnostics.Redact(err.Error()))
}

func errOp(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) && ae != nil && ae.Op != "" {
		return ae.Op
	}
	return "registry"
}

func errSubject(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) && ae != nil && ae.Subject != "" {
		return ae.Subject
	}
	return ""
}
