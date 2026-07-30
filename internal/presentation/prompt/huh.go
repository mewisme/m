package prompt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	huh "charm.land/huh/v2"

	"github.com/mewisme/mew/internal/apperr"
	mewprompt "github.com/mewisme/mew/internal/prompt"
)

// HuhPrompter renders rich prompts via Huh v2 on stderr/stdin.
type HuhPrompter struct {
	in       io.Reader
	errW     io.Writer
	width    int
	useColor bool
}

// NewHuh builds a rich Huh-backed Prompter.
func NewHuh(opts Options) *HuhPrompter {
	in := opts.Stdin
	errW := opts.Stderr
	if errW == nil {
		errW = io.Discard
	}
	w := opts.Width
	if w <= 0 {
		w = 80
	}
	return &HuhPrompter{in: in, errW: errW, width: w, useColor: opts.UseColor}
}

// Prompt runs a Huh form. Cancellation maps to Cancelled; EOF uses SafeDefaultID.
func (p *HuhPrompter) Prompt(ctx context.Context, req mewprompt.PromptRequest) (mewprompt.PromptAnswer, error) {
	if err := ctx.Err(); err != nil {
		return mewprompt.PromptAnswer{Cancelled: true}, err
	}
	if err := mewprompt.ValidateRequest(req); err != nil {
		return mewprompt.PromptAnswer{}, err
	}
	req = normalizeConfirm(req)

	var fields []huh.Field
	noteField := buildNote(req)
	if noteField != nil {
		fields = append(fields, noteField)
	}

	switch req.Kind {
	case mewprompt.PromptInput:
		var value string
		input := huh.NewInput().Title(firstNonEmpty(req.Title, "Input")).Value(&value)
		if req.Secret {
			input = input.EchoMode(huh.EchoModePassword)
		}
		fields = append(fields, input)
		form := p.form(fields...)
		if err := form.RunWithContext(ctx); err != nil {
			return mapHuhErr(req, err)
		}
		if req.Secret {
			value = ""
		}
		return mewprompt.PromptAnswer{Value: value}, nil

	case mewprompt.PromptConfirm, mewprompt.PromptSelect:
		selected := mewprompt.SafeDefaultID(req)
		if selected == "" && len(req.Options) > 0 {
			selected = req.Options[0].ID
		}
		opts := make([]huh.Option[string], 0, len(req.Options))
		for _, o := range req.Options {
			opts = append(opts, huh.NewOption(o.Label, o.ID))
		}
		sel := huh.NewSelect[string]().
			Title(firstNonEmpty(req.Title, "Choose")).
			Options(opts...).
			Value(&selected)
		if req.Description != "" && noteField == nil {
			sel = sel.Description(req.Description)
		}
		fields = append(fields, sel)
		form := p.form(fields...)
		if err := form.RunWithContext(ctx); err != nil {
			return mapHuhErr(req, err)
		}
		return mewprompt.PromptAnswer{OptionID: selected}, nil

	default:
		return mewprompt.PromptAnswer{}, &mewprompt.ValidationError{Message: "unsupported prompt kind"}
	}
}

func (p *HuhPrompter) form(fields ...huh.Field) *huh.Form {
	form := huh.NewForm(huh.NewGroup(fields...)).
		WithWidth(p.width).
		WithOutput(p.errW).
		WithShowHelp(false)
	if p.in != nil {
		form = form.WithInput(p.in)
	}
	if p.useColor {
		form = form.WithTheme(huh.ThemeFunc(huh.ThemeBase16))
	} else {
		form = form.WithTheme(huh.ThemeFunc(huh.ThemeBase))
	}
	return form
}

func buildNote(req mewprompt.PromptRequest) huh.Field {
	var b strings.Builder
	if req.Description != "" && req.Kind == mewprompt.PromptInput {
		b.WriteString(req.Description)
	}
	for _, f := range req.Fields {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s: %s", f.Key, f.Value)
	}
	// For select/confirm, put description into the note when fields exist,
	// otherwise Description is attached to the select field.
	if req.Kind != mewprompt.PromptInput && len(req.Fields) > 0 && req.Description != "" {
		prefix := req.Description + "\n"
		return huh.NewNote().Title("").Description(prefix + b.String())
	}
	if b.Len() == 0 {
		return nil
	}
	return huh.NewNote().Title("").Description(b.String())
}

func mapHuhErr(req mewprompt.PromptRequest, err error) (mewprompt.PromptAnswer, error) {
	if err == nil {
		return mewprompt.PromptAnswer{}, nil
	}
	if errors.Is(err, huh.ErrUserAborted) || errors.Is(err, context.Canceled) {
		return mewprompt.PromptAnswer{Cancelled: true}, mewprompt.ErrCancelled
	}
	if errors.Is(err, io.EOF) {
		return eofAnswer(req), nil
	}
	return mewprompt.PromptAnswer{}, apperr.Wrap(apperr.IO, "prompt.huh", req.ID, err)
}

var _ mewprompt.Prompter = (*HuhPrompter)(nil)
