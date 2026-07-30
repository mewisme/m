// Package prompt implements Huh-backed and accessible numbered Prompters.
// Domain packages must not import this package; they use internal/prompt only.
package prompt

import (
	"context"
	"io"

	mewprompt "github.com/mewisme/mew/internal/prompt"
)

// Options configures prompt adapters for one invocation.
type Options struct {
	Stdin  io.Reader
	Stderr io.Writer
	Width  int
	// UseColor enables Huh theming; false forces ThemeBase.
	UseColor bool
	// UseUnicode is reserved for future symbol selection in summaries.
	UseUnicode bool
	// Accessible forces the numbered append-only adapter.
	Accessible bool
	// UseRich selects Huh when Accessible is false.
	UseRich bool
	// Suspend pauses live UI before Prompt; Resume restores afterward.
	Suspend func(context.Context) error
	Resume  func(context.Context) error
}

// New selects Huh or Accessible and optionally wraps Suspend/Resume.
func New(opts Options) mewprompt.Prompter {
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Width <= 0 {
		opts.Width = 80
	}
	var inner mewprompt.Prompter
	if opts.Accessible || !opts.UseRich {
		inner = NewAccessible(opts)
	} else {
		inner = NewHuh(opts)
	}
	if opts.Suspend != nil || opts.Resume != nil {
		return &suspendPrompter{inner: inner, suspend: opts.Suspend, resume: opts.Resume}
	}
	return inner
}

type suspendPrompter struct {
	inner   mewprompt.Prompter
	suspend func(context.Context) error
	resume  func(context.Context) error
}

func (s *suspendPrompter) Prompt(ctx context.Context, req mewprompt.PromptRequest) (mewprompt.PromptAnswer, error) {
	if s.suspend != nil {
		if err := s.suspend(ctx); err != nil {
			return mewprompt.PromptAnswer{}, err
		}
	}
	if s.resume != nil {
		defer func() { _ = s.resume(ctx) }()
	}
	return s.inner.Prompt(ctx, req)
}

var _ mewprompt.Prompter = (*suspendPrompter)(nil)
