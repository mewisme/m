// Package pager resolves and executes an optional safe pager for topic help.
package pager

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode"

	"github.com/mewisme/mew/internal/apperr"
)

// Mode is the pager selection mode.
type Mode string

const (
	ModeAuto   Mode = "auto"
	ModeAlways Mode = "always"
	ModeNever  Mode = "never"
)

// AutoLineThreshold triggers auto paging when content has at least this many lines.
const AutoLineThreshold = 24

// Input is the resolution input for one help invocation.
type Input struct {
	Flag        string // --pager value (auto|always|never or empty)
	MEWPager    string
	ConfigPager string
	PAGER       string
	StdoutTTY   bool
	Human       bool
	CI          bool
	Accessible  bool
	LineCount   int
}

// Plan is the resolved pager action.
type Plan struct {
	Mode    Mode
	Use     bool
	Path    string
	Args    []string
	Command string // original command string for diagnostics
}

// Resolve decides whether and how to page content.
func Resolve(in Input) (Plan, error) {
	mode, err := parseMode(in.Flag)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Mode: mode}
	if mode == ModeNever {
		return plan, nil
	}

	cmdStr := firstNonEmpty(in.MEWPager, in.ConfigPager, in.PAGER)
	if cmdStr == "" {
		// Windows: never assume less. Unix auto may use less when present.
		if runtime.GOOS != "windows" {
			cmdStr = "less"
		}
	}
	if cmdStr == "" {
		if mode == ModeAlways {
			return Plan{}, apperr.New(apperr.IO, "help.pager", "", "pager required but no pager command is configured")
		}
		return plan, nil
	}

	path, args, err := SplitCommand(cmdStr)
	if err != nil {
		return Plan{}, err
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		if mode == ModeAlways {
			return Plan{}, apperr.Wrap(apperr.IO, "help.pager", path, err)
		}
		return plan, nil
	}
	plan.Path = resolved
	plan.Args = args
	plan.Command = cmdStr

	switch mode {
	case ModeAlways:
		plan.Use = true
	case ModeAuto:
		plan.Use = in.StdoutTTY && in.Human && !in.CI && (!in.Accessible) && in.LineCount >= AutoLineThreshold
	}
	return plan, nil
}

func parseMode(flag string) (Mode, error) {
	flag = strings.TrimSpace(strings.ToLower(flag))
	switch flag {
	case "", "auto":
		return ModeAuto, nil
	case "always":
		return ModeAlways, nil
	case "never":
		return ModeNever, nil
	default:
		return "", apperr.New(apperr.Usage, "help.pager", flag, "pager must be auto, always, or never")
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// SplitCommand parses a pager command into argv without a shell.
// Supports simple quoting with single or double quotes.
func SplitCommand(s string) (string, []string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil, apperr.New(apperr.Usage, "help.pager", "", "empty pager command")
	}
	if strings.ContainsAny(s, "|&;<>$`(){}!\n\r") {
		return "", nil, apperr.New(apperr.Usage, "help.pager", s, "pager command contains unsafe shell metacharacters")
	}
	var parts []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		parts = append(parts, cur.String())
		cur.Reset()
	}
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if quote != 0 {
		return "", nil, apperr.New(apperr.Usage, "help.pager", s, "unclosed quote in pager command")
	}
	flush()
	if len(parts) == 0 {
		return "", nil, apperr.New(apperr.Usage, "help.pager", s, "empty pager command")
	}
	return parts[0], parts[1:], nil
}

// WritePaged writes content through a pager when plan.Use is set.
// Auto-mode pager failures fall back to direct writes. Always-mode failures are typed I/O errors.
func WritePaged(ctx context.Context, w io.Writer, content string, plan Plan) error {
	if !plan.Use || plan.Path == "" {
		_, err := io.WriteString(w, content)
		return err
	}
	cmd := exec.CommandContext(ctx, plan.Path, plan.Args...)
	cmd.Stdout = w
	if f, ok := w.(*os.File); ok {
		cmd.Stdout = f
	}
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		if plan.Mode == ModeAlways {
			return apperr.Wrap(apperr.IO, "help.pager", plan.Path, err)
		}
		_, werr := io.WriteString(w, content)
		return werr
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		if plan.Mode == ModeAlways {
			return apperr.Wrap(apperr.IO, "help.pager", plan.Path, err)
		}
		_, werr := io.WriteString(w, content)
		return werr
	}
	_, writeErr := io.WriteString(stdin, content)
	_ = stdin.Close()
	waitErr := cmd.Wait()
	if writeErr != nil && isBrokenPipe(writeErr) {
		// Pager exited early; treat as success for auto and always (user quit).
		return nil
	}
	if waitErr != nil {
		if plan.Mode == ModeAlways && !isExitError(waitErr) {
			return apperr.Wrap(apperr.IO, "help.pager", plan.Path, waitErr)
		}
		// Non-zero pager exit (e.g. less quit) is normal.
		if writeErr == nil {
			return nil
		}
	}
	if writeErr != nil && plan.Mode == ModeAlways {
		return apperr.Wrap(apperr.IO, "help.pager", plan.Path, writeErr)
	}
	if writeErr != nil && plan.Mode == ModeAuto {
		// Content may have been partially consumed; do not rewrite (would duplicate).
		return nil
	}
	return nil
}

func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "pipe is being closed")
}

func isExitError(err error) bool {
	_, ok := err.(*exec.ExitError)
	return ok
}

// LineCount returns the number of lines in s.
func LineCount(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// FormatMissing explains auto fallback (test helper surface).
func FormatMissing(cmd string) string {
	return fmt.Sprintf("pager %q unavailable; writing directly", cmd)
}
