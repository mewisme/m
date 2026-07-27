package apperr

import (
	"errors"
	"fmt"
)

// OperationFailure pairs a primary operation error with optional cleanup failures.
type OperationFailure struct {
	Primary error
	Cleanup error
}

func (e *OperationFailure) Error() string {
	if e == nil {
		return ""
	}
	if e.Cleanup == nil {
		if e.Primary != nil {
			return e.Primary.Error()
		}
		return ""
	}
	if e.Primary == nil {
		return fmt.Sprintf("cleanup: %s", e.Cleanup.Error())
	}
	return fmt.Sprintf("%s (cleanup: %s)", e.Primary.Error(), e.Cleanup.Error())
}

func (e *OperationFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Primary
}

func (e *OperationFailure) Is(target error) bool {
	if e == nil {
		return false
	}
	return errors.Is(e.Primary, target) || errors.Is(e.Cleanup, target)
}

// WithCleanup returns primary joined with cleanup errors as OperationFailure.
func WithCleanup(primary error, cleanup ...error) error {
	joined := errors.Join(cleanup...)
	if primary == nil {
		return joined
	}
	if joined == nil {
		return primary
	}
	return &OperationFailure{Primary: primary, Cleanup: joined}
}

// JoinCleanup returns primary paired with a single cleanup error.
func JoinCleanup(primary, cleanupErr error) error {
	if cleanupErr == nil {
		return primary
	}
	return WithCleanup(primary, cleanupErr)
}
