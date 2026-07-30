package fsx

import (
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
)

// RenameError records a failed filesystem rename with source and destination paths.
type RenameError struct {
	Op    string
	Src   string
	Dst   string
	Cause error
}

func (e *RenameError) Error() string {
	if e == nil {
		return ""
	}
	op := e.Op
	if op == "" {
		op = "rename"
	}
	msg := ""
	if e.Cause != nil {
		msg = e.Cause.Error()
	}
	if msg == "" {
		return fmt.Sprintf("%s %s -> %s", op, e.Src, e.Dst)
	}
	return fmt.Sprintf("%s %s -> %s: %s", op, e.Src, e.Dst, msg)
}

func (e *RenameError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewRenameError builds a rename error with operation metadata.
func NewRenameError(op, src, dst string, cause error) *RenameError {
	return &RenameError{Op: op, Src: src, Dst: dst, Cause: cause}
}

// WrapPublishRename wraps a rename failure for transaction/fsx publish paths.
func WrapPublishRename(op, src, dst string, err error) error {
	if err == nil {
		return nil
	}
	var re *RenameError
	if !AsRenameError(err, &re) {
		re = NewRenameError("rename", src, dst, err)
	}
	return apperr.Wrap(apperr.IO, op, dst, re)
}

// AsRenameError reports whether err is or wraps a RenameError.
func AsRenameError(err error, target **RenameError) bool {
	if err == nil {
		return false
	}
	if re, ok := err.(*RenameError); ok {
		*target = re
		return true
	}
	if u, ok := err.(interface{ Unwrap() error }); ok {
		return AsRenameError(u.Unwrap(), target)
	}
	return false
}
