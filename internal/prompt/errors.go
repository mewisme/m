package prompt

import (
	"errors"
	"fmt"
)

// ErrCancelled indicates the user aborted the prompt (Ctrl+C).
var ErrCancelled = errors.New("prompt cancelled")

// UnavailableError reports that a required prompt cannot run.
type UnavailableError struct {
	NeedTTY bool
	Reason  string
}

func (e *UnavailableError) Error() string {
	if e == nil {
		return "prompt unavailable"
	}
	if e.Reason != "" {
		return e.Reason
	}
	if e.NeedTTY {
		return "interactive prompt requires a TTY on stdin"
	}
	return "prompt unavailable"
}

// ValidationError reports an invalid PromptRequest.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil || e.Message == "" {
		return "invalid prompt request"
	}
	return e.Message
}

// ValidateRequest checks option IDs and kind constraints before rendering.
func ValidateRequest(req PromptRequest) error {
	switch req.Kind {
	case PromptConfirm, PromptSelect, PromptInput:
	default:
		return &ValidationError{Message: fmt.Sprintf("unknown prompt kind %q", req.Kind)}
	}
	seen := map[string]struct{}{}
	for _, opt := range req.Options {
		id := opt.ID
		if id == "" {
			return &ValidationError{Message: "empty option id"}
		}
		if _, ok := seen[id]; ok {
			return &ValidationError{Message: fmt.Sprintf("duplicate option id %q", id)}
		}
		seen[id] = struct{}{}
	}
	if req.Kind == PromptSelect && len(req.Options) == 0 {
		return &ValidationError{Message: "select prompt requires options"}
	}
	// Confirm may omit options; adapters synthesize approve/reject.
	if req.DefaultID != "" && len(req.Options) > 0 {
		if _, ok := seen[req.DefaultID]; !ok {
			return &ValidationError{Message: fmt.Sprintf("default id %q not in options", req.DefaultID)}
		}
	}
	return nil
}
