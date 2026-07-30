// Package prompt defines the stdlib-only interactive prompt contract.
// Domain and app packages may import this package. Charm and presentation
// adapters live under internal/presentation/prompt.
package prompt

import "context"

// PromptKind selects the interaction shape.
type PromptKind string

const (
	PromptConfirm PromptKind = "confirm"
	PromptSelect  PromptKind = "select"
	PromptInput   PromptKind = "input"
)

// Stable option IDs interpreted by domain policy.
const (
	OptionDeny         = "deny"
	OptionAllowOnce    = "allow-once"
	OptionTrustProject = "trust-project"
	OptionApprove      = "approve"
	OptionReject       = "reject"
)

// Field is an already-redacted display key/value pair.
type Field struct {
	Key   string
	Value string
}

// Option is one selectable choice with a stable ID.
type Option struct {
	ID    string
	Label string
}

// PromptRequest describes one user prompt. Display fields must already be redacted.
type PromptRequest struct {
	ID          string
	Kind        PromptKind
	Title       string
	Description string
	Fields      []Field
	Options     []Option
	DefaultID   string
	Dangerous   bool
	Secret      bool
}

// PromptAnswer is the user response. Cancelled is set for Ctrl+C / abort.
type PromptAnswer struct {
	OptionID  string
	Value     string
	Cancelled bool
}

// Prompter renders a prompt and returns the answer. Adapters must not mutate
// trust stores or perform product side effects.
type Prompter interface {
	Prompt(ctx context.Context, req PromptRequest) (PromptAnswer, error)
}
