package prompt

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	mewprompt "github.com/mewisme/mew/internal/prompt"
)

const maxInvalidSelections = 5

// AccessiblePrompter is an append-only numbered prompt (no cursor repaint).
type AccessiblePrompter struct {
	in    io.Reader
	errW  io.Writer
	width int
}

// NewAccessible builds a line-oriented numbered Prompter.
func NewAccessible(opts Options) *AccessiblePrompter {
	in := opts.Stdin
	if in == nil {
		in = strings.NewReader("")
	}
	errW := opts.Stderr
	if errW == nil {
		errW = io.Discard
	}
	w := opts.Width
	if w <= 0 {
		w = 80
	}
	return &AccessiblePrompter{in: in, errW: errW, width: w}
}

// Prompt writes numbered choices to stderr and reads a selection from stdin.
func (p *AccessiblePrompter) Prompt(ctx context.Context, req mewprompt.PromptRequest) (mewprompt.PromptAnswer, error) {
	if err := ctx.Err(); err != nil {
		return mewprompt.PromptAnswer{Cancelled: true}, err
	}
	if err := mewprompt.ValidateRequest(req); err != nil {
		return mewprompt.PromptAnswer{}, err
	}
	req = normalizeConfirm(req)

	switch req.Kind {
	case mewprompt.PromptInput:
		return p.promptInput(ctx, req)
	case mewprompt.PromptConfirm, mewprompt.PromptSelect:
		return p.promptSelect(ctx, req)
	default:
		return mewprompt.PromptAnswer{}, &mewprompt.ValidationError{Message: "unsupported prompt kind"}
	}
}

func (p *AccessiblePrompter) promptSelect(ctx context.Context, req mewprompt.PromptRequest) (mewprompt.PromptAnswer, error) {
	p.writeHeader(req)
	_, _ = fmt.Fprintln(p.errW, "Choose an action:")
	for i, opt := range req.Options {
		_, _ = fmt.Fprintf(p.errW, "%d. %s\n", i+1, opt.Label)
	}
	defIdx := defaultIndex(req)
	promptLine := "Selection"
	if defIdx >= 0 {
		promptLine = fmt.Sprintf("Selection [%d]", defIdx+1)
	}
	reader := bufio.NewReader(p.in)
	for attempt := 0; attempt < maxInvalidSelections; attempt++ {
		if err := ctx.Err(); err != nil {
			return mewprompt.PromptAnswer{Cancelled: true}, err
		}
		_, _ = fmt.Fprintf(p.errW, "%s: ", promptLine)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return eofAnswer(req), nil
			}
			return mewprompt.PromptAnswer{}, apperr.Wrap(apperr.IO, "prompt.accessible", req.ID, err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return emptyEnterAnswer(req), nil
		}
		n, perr := strconv.Atoi(line)
		if perr != nil || n < 1 || n > len(req.Options) {
			_, _ = fmt.Fprintln(p.errW, "Invalid selection; enter a listed number.")
			continue
		}
		return mewprompt.PromptAnswer{OptionID: req.Options[n-1].ID}, nil
	}
	return eofAnswer(req), nil
}

func (p *AccessiblePrompter) promptInput(ctx context.Context, req mewprompt.PromptRequest) (mewprompt.PromptAnswer, error) {
	p.writeHeader(req)
	_, _ = fmt.Fprintf(p.errW, "%s: ", firstNonEmpty(req.Title, "Input"))
	if err := ctx.Err(); err != nil {
		return mewprompt.PromptAnswer{Cancelled: true}, err
	}
	line, err := bufio.NewReader(p.in).ReadString('\n')
	if err != nil && err != io.EOF {
		return mewprompt.PromptAnswer{}, apperr.Wrap(apperr.IO, "prompt.accessible", req.ID, err)
	}
	if err == io.EOF && strings.TrimSpace(line) == "" {
		return eofAnswer(req), nil
	}
	val := strings.TrimSpace(line)
	if req.Secret {
		val = ""
	}
	return mewprompt.PromptAnswer{Value: val, OptionID: mewprompt.SafeDefaultID(req)}, nil
}

func (p *AccessiblePrompter) writeHeader(req mewprompt.PromptRequest) {
	if req.Title != "" {
		_, _ = fmt.Fprintln(p.errW, wrapASCII(req.Title, p.width))
	}
	if req.Description != "" {
		_, _ = fmt.Fprintln(p.errW, wrapASCII(req.Description, p.width))
	}
	for _, f := range req.Fields {
		_, _ = fmt.Fprintf(p.errW, "%s: %s\n", f.Key, f.Value)
	}
}

func normalizeConfirm(req mewprompt.PromptRequest) mewprompt.PromptRequest {
	if req.Kind != mewprompt.PromptConfirm {
		return req
	}
	if len(req.Options) > 0 {
		return req
	}
	req.Options = []mewprompt.Option{
		{ID: mewprompt.OptionReject, Label: "No"},
		{ID: mewprompt.OptionApprove, Label: "Yes"},
	}
	if req.DefaultID == "" {
		req.DefaultID = mewprompt.OptionReject
	}
	return req
}

func defaultIndex(req mewprompt.PromptRequest) int {
	id := mewprompt.SafeDefaultID(req)
	if id == "" {
		id = req.DefaultID
		if req.Dangerous {
			switch id {
			case mewprompt.OptionDeny, mewprompt.OptionReject:
			default:
				return -1
			}
		}
	}
	for i, opt := range req.Options {
		if opt.ID == id {
			return i
		}
	}
	return -1
}

func eofAnswer(req mewprompt.PromptRequest) mewprompt.PromptAnswer {
	id := mewprompt.SafeDefaultID(req)
	if id == "" {
		return mewprompt.PromptAnswer{Cancelled: true}
	}
	return mewprompt.PromptAnswer{OptionID: id}
}

func emptyEnterAnswer(req mewprompt.PromptRequest) mewprompt.PromptAnswer {
	return eofAnswer(req)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func wrapASCII(s string, width int) string {
	s = strings.TrimSpace(s)
	if width < 20 {
		width = 20
	}
	if len(s) <= width {
		return s
	}
	var b strings.Builder
	for len(s) > width {
		b.WriteString(s[:width])
		b.WriteByte('\n')
		s = s[width:]
	}
	b.WriteString(s)
	return b.String()
}

var _ mewprompt.Prompter = (*AccessiblePrompter)(nil)
