package presentation

import (
	"errors"
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// MapOptions controls ErrorView mapping.
type MapOptions struct {
	Debug      bool
	Redact     func(string) string
	BinaryName string // "m", "mew", "mx", "mewx"
}

func (o MapOptions) redact(s string) string {
	if o.Redact == nil {
		return s
	}
	return o.Redact(s)
}

// MapError maps a typed error to an ErrorView without styling.
func MapError(err error, opts MapOptions) ErrorView {
	if err == nil {
		return ErrorView{}
	}
	meta, view := mapPrimaryError(err, opts)
	view.Hints = HintsFor(meta, opts.Debug, opts.BinaryName)
	if opts.Debug {
		view.Causes = collectCauses(err, opts)
	}
	return view
}

func mapPrimaryError(err error, opts MapOptions) (ErrorMetadata, ErrorView) {
	var of *apperr.OperationFailure
	if errors.As(err, &of) && of != nil && of.Primary != nil {
		err = of.Primary
	}
	code := apperr.CodeOf(err)
	meta := ErrorMetadata{Code: code}
	view := ErrorView{
		Severity: StatusError,
		Title:    TitleForCode(code),
		Code:     string(code),
	}
	var ae *apperr.Error
	if errors.As(err, &ae) && ae != nil {
		meta.Operation = ae.Op
		meta.Subject = ae.Subject
		view.Operation = opts.redact(ae.Op)
		view.Subject = opts.redact(ae.Subject)
		view.Message = opts.redact(errorMessage(ae))
		view.Context = appendContextFromError(view.Context, ae, opts)
		return meta, view
	}
	view.Message = opts.redact(err.Error())
	return meta, view
}

func errorMessage(ae *apperr.Error) string {
	if ae == nil {
		return ""
	}
	if ae.Message != "" {
		return ae.Message
	}
	if ae.Cause != nil {
		return ae.Cause.Error()
	}
	return ae.Error()
}

func appendContextFromError(ctx []KeyValue, ae *apperr.Error, opts MapOptions) []KeyValue {
	if ae == nil {
		return ctx
	}
	if ae.Subject != "" {
		ctx = append(ctx, KeyValue{Key: "Subject", Value: opts.redact(ae.Subject)})
	}
	if ae.Op != "" {
		ctx = append(ctx, KeyValue{Key: "Operation", Value: opts.redact(ae.Op)})
	}
	var renameErr *fsx.RenameError
	if errors.As(ae.Cause, &renameErr) && renameErr != nil {
		if renameErr.Op != "" {
			ctx = append(ctx, KeyValue{Key: "Filesystem op", Value: opts.redact(renameErr.Op)})
		}
		if renameErr.Src != "" {
			ctx = append(ctx, KeyValue{Key: "Source", Value: opts.redact(renameErr.Src), Style: ValuePath})
		}
		if renameErr.Dst != "" {
			ctx = append(ctx, KeyValue{Key: "Destination", Value: opts.redact(renameErr.Dst), Style: ValuePath})
		}
	}
	return ctx
}

func collectCauses(err error, opts MapOptions) []CauseView {
	var out []CauseView
	for cur := err; cur != nil; {
		var ae *apperr.Error
		if errors.As(cur, &ae) && ae != nil && ae.Cause != nil {
			out = append(out, CauseView{
				Label:   "cause",
				Message: opts.redact(ae.Cause.Error()),
			})
			cur = ae.Cause
			continue
		}
		cur = errors.Unwrap(cur)
		if cur == nil {
			break
		}
		out = append(out, CauseView{
			Label:   "cause",
			Message: opts.redact(cur.Error()),
		})
		if len(out) >= 5 {
			break
		}
	}
	return out
}

// FormatErrorCode returns a display line for the error code field.
func FormatErrorCode(code string) string {
	if code == "" {
		return ""
	}
	return fmt.Sprintf("Code: %s", code)
}
