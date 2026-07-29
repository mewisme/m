// Package apperr defines typed Mew errors with stable ERR_M_* codes.
package apperr

import (
	"errors"
	"fmt"
)

// Code is a stable machine-readable error identifier.
type Code string

const (
	OK                  Code = "ERR_M_OK"
	Usage               Code = "ERR_M_USAGE"
	Cancelled           Code = "ERR_M_CANCELLED"
	Internal            Code = "ERR_M_INTERNAL"
	InternalPanic       Code = "ERR_M_INTERNAL_PANIC"
	IO                  Code = "ERR_M_IO"
	Config              Code = "ERR_M_CONFIG"
	Network             Code = "ERR_M_NETWORK"
	Integrity           Code = "ERR_M_INTEGRITY"
	Lockfile            Code = "ERR_M_LOCKFILE"
	LockUnsupported     Code = "ERR_M_LOCK_UNSUPPORTED"
	LockAmbiguous       Code = "ERR_M_LOCK_AMBIGUOUS"
	LockUnrepresentable Code = "ERR_M_LOCK_UNREPRESENTABLE"
	Unimplemented       Code = "ERR_M_UNIMPLEMENTED"
	Unsupported         Code = "ERR_M_UNSUPPORTED"
	Manifest            Code = "ERR_M_MANIFEST"
	NotFound            Code = "ERR_M_NOT_FOUND"
	Resolve             Code = "ERR_M_RESOLVE"
	Install             Code = "ERR_M_INSTALL"
	Transaction         Code = "ERR_M_TRANSACTION"
	Store               Code = "ERR_M_STORE"
	Policy              Code = "ERR_M_POLICY"
	PNPUnsupported      Code = "ERR_M_PNP_UNSUPPORTED"
)

// registry maps every published code to a process exit status.
var registry = map[Code]int{
	OK:                  0,
	Usage:               2,
	Cancelled:           130,
	Internal:            1,
	InternalPanic:       1,
	IO:                  1,
	Config:              1,
	Network:             1,
	Integrity:           1,
	Lockfile:            1,
	LockUnsupported:     1,
	LockAmbiguous:       1,
	LockUnrepresentable: 1,
	Unimplemented:       1,
	Unsupported:         1,
	Manifest:            1,
	NotFound:            1,
	Resolve:             1,
	Install:             1,
	Transaction:         1,
	Store:               1,
	Policy:              1,
	PNPUnsupported:      1,
}

// AllCodes returns registered codes in a stable order for docs and tests.
func AllCodes() []Code {
	return []Code{
		OK, Usage, Cancelled, Internal, InternalPanic,
		IO, Config, Network, Integrity, Lockfile, LockUnsupported, LockAmbiguous, LockUnrepresentable, Unimplemented, Unsupported,
		Manifest, NotFound, Resolve, Install, Transaction, Store, Policy, PNPUnsupported,
	}
}

// ExitForCode returns the exit status for a registered code, or 1 if unknown.
func ExitForCode(c Code) int {
	if n, ok := registry[c]; ok {
		return n
	}
	return 1
}

// Error is a typed application error.
type Error struct {
	Code     Code
	Op       string
	Subject  string
	Message  string
	Cause    error
	ExitHint int // 0 = use Code mapping
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Message
	if msg == "" && e.Cause != nil {
		msg = e.Cause.Error()
	}
	if e.Op != "" && e.Subject != "" {
		return fmt.Sprintf("%s: %s: %s: %s", e.Code, e.Op, e.Subject, msg)
	}
	if e.Op != "" {
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Op, msg)
	}
	return fmt.Sprintf("%s: %s", e.Code, msg)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// New constructs a typed error without a cause.
func New(code Code, op, subject, msg string) *Error {
	return &Error{Code: code, Op: op, Subject: subject, Message: msg}
}

// Wrap constructs a typed error wrapping cause.
func Wrap(code Code, op, subject string, err error) *Error {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return &Error{Code: code, Op: op, Subject: subject, Message: msg, Cause: err}
}

// CodeOf extracts the first apperr.Code in the chain, or Internal if none.
// OperationFailure resolves through Primary before falling back to Internal.
func CodeOf(err error) Code {
	var of *OperationFailure
	if errors.As(err, &of) && of != nil && of.Primary != nil {
		err = of.Primary
	}
	var ae *Error
	if errors.As(err, &ae) && ae != nil && ae.Code != "" {
		return ae.Code
	}
	return Internal
}

// ExitCode maps an error to a process exit status.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ae *Error
	if errors.As(err, &ae) && ae != nil {
		if ae.ExitHint != 0 {
			return ae.ExitHint
		}
		return ExitForCode(ae.Code)
	}
	return 1
}

// IsUsage reports whether err is a usage / invalid-arguments failure.
func IsUsage(err error) bool {
	return CodeOf(err) == Usage
}
