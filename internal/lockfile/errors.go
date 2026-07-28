package lockfile

import (
	"github.com/mewisme/mew/internal/apperr"
)

// RepresentabilityError carries a loss report for fail-closed encode paths.
type RepresentabilityError struct {
	Err    *apperr.Error
	Report LossReport
}

func (e *RepresentabilityError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *RepresentabilityError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewUnsupported reports an unsupported lockfile generation.
func NewUnsupported(op, subject, msg string) *apperr.Error {
	return apperr.New(apperr.LockUnsupported, op, subject, msg)
}

// NewAmbiguous reports ambiguous producer-major detection.
func NewAmbiguous(op, subject, msg string) *apperr.Error {
	return apperr.New(apperr.LockAmbiguous, op, subject, msg)
}

// NewUnrepresentable reports a lossy lock encode with a loss report.
func NewUnrepresentable(op, subject, msg string, report LossReport) *RepresentabilityError {
	return &RepresentabilityError{
		Err:    apperr.New(apperr.LockUnrepresentable, op, subject, msg),
		Report: report,
	}
}
