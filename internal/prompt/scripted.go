package prompt

import (
	"context"
	"io"
)

// ScriptedPrompter returns predetermined answers for tests.
type ScriptedPrompter struct {
	Answers []PromptAnswer
	Errs    []error
	Calls   []PromptRequest
	i       int
}

// Prompt records the request and returns the next scripted answer.
func (s *ScriptedPrompter) Prompt(ctx context.Context, req PromptRequest) (PromptAnswer, error) {
	if err := ctx.Err(); err != nil {
		return PromptAnswer{Cancelled: true}, err
	}
	if err := ValidateRequest(req); err != nil {
		return PromptAnswer{}, err
	}
	s.Calls = append(s.Calls, req)
	idx := s.i
	s.i++
	if idx < len(s.Errs) && s.Errs[idx] != nil {
		return PromptAnswer{}, s.Errs[idx]
	}
	if idx < len(s.Answers) {
		return s.Answers[idx], nil
	}
	// Fail closed when the script is exhausted.
	return PromptAnswer{OptionID: SafeDefaultID(req), Cancelled: SafeDefaultID(req) == ""}, io.EOF
}

var _ Prompter = (*ScriptedPrompter)(nil)
