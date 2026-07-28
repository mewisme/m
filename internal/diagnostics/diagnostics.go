// Package diagnostics provides redaction, progress events, and reporters.
package diagnostics

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mewisme/mew/internal/apperr"
)

// Attr is a debug key/value pair.
type Attr struct {
	Key   string
	Value string
}

// Event is a progress notification independent of terminal rendering.
type Event struct {
	V          int     `json:"v"`
	Type       string  `json:"type"`
	Phase      string  `json:"phase"`
	Package    string  `json:"package,omitempty"`
	Bytes      int64   `json:"bytes,omitempty"`
	TotalBytes *int64  `json:"total_bytes"`
	OpID       string  `json:"op_id,omitempty"`
	TxID       *string `json:"tx_id"`
}

// ErrorDocument is the JSON reporter error payload.
type ErrorDocument struct {
	V       int    `json:"v"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Op      string `json:"op,omitempty"`
	Subject string `json:"subject,omitempty"`
	Message string `json:"message"`
	Exit    int    `json:"exit"`
}

// Reporter emits progress, errors, and debug lines.
type Reporter interface {
	Progress(Event)
	Error(err error)
	Debug(msg string, attrs ...Attr)
}

// Options configures a reporter.
type Options struct {
	Out       io.Writer // progress / json stdout (default os.Stdout)
	Err       io.Writer // errors (default os.Stderr)
	Format    string    // default | ndjson | json | silent
	Color     ColorMode
	Debug     bool
	Unsafe    bool // skip redaction (requires explicit flag)
	TermWidth int  // 0 = default 80
}

// ColorMode controls ANSI color.
type ColorMode int

const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

// NewReporter builds a reporter for the given format.
func NewReporter(opts Options) Reporter {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}
	if opts.TermWidth <= 0 {
		opts.TermWidth = 80
	}
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" || format == "human" {
		format = "default"
	}
	base := &baseReporter{opts: opts}
	switch format {
	case "ndjson":
		return &ndjsonReporter{base: base}
	case "json":
		return &jsonReporter{base: base}
	case "silent":
		return &silentReporter{base: base}
	default:
		return &humanReporter{base: base}
	}
}

type baseReporter struct {
	opts Options
	mu   sync.Mutex
}

func (b *baseReporter) redact(s string) string {
	if b.opts.Unsafe {
		return s
	}
	return Redact(s)
}

func (b *baseReporter) colorEnabled(w io.Writer) bool {
	switch b.opts.Color {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		return isTTY(w)
	}
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

type humanReporter struct{ base *baseReporter }

func (r *humanReporter) Progress(ev Event) {
	r.base.mu.Lock()
	defer r.base.mu.Unlock()
	line := ev.Phase
	if ev.Package != "" {
		line += " " + ev.Package
	}
	if ev.Bytes > 0 {
		line += fmt.Sprintf(" %dB", ev.Bytes)
	}
	line = truncate(r.base.redact(line), r.base.opts.TermWidth)
	fmt.Fprintln(r.base.opts.Err, line)
}

func (r *humanReporter) Error(err error) {
	r.base.mu.Lock()
	defer r.base.mu.Unlock()
	msg := truncate(r.base.redact(formatHumanError(err)), r.base.opts.TermWidth)
	if r.base.colorEnabled(r.base.opts.Err) {
		fmt.Fprintf(r.base.opts.Err, "\x1b[31m%s\x1b[0m\n", msg)
		return
	}
	fmt.Fprintln(r.base.opts.Err, msg)
}

func (r *humanReporter) Debug(msg string, attrs ...Attr) {
	if !r.base.opts.Debug {
		return
	}
	r.base.mu.Lock()
	defer r.base.mu.Unlock()
	line := "debug: " + r.base.redact(msg)
	for _, a := range attrs {
		line += fmt.Sprintf(" %s=%s", a.Key, r.base.redact(a.Value))
	}
	fmt.Fprintln(r.base.opts.Err, truncate(line, r.base.opts.TermWidth))
}

type ndjsonReporter struct{ base *baseReporter }

func (r *ndjsonReporter) Progress(ev Event) {
	if ev.V == 0 {
		ev.V = 1
	}
	if ev.Type == "" {
		ev.Type = "progress"
	}
	ev.Phase = r.base.redact(ev.Phase)
	ev.Package = r.base.redact(ev.Package)
	ev.OpID = r.base.redact(ev.OpID)
	r.writeLine(ev)
}

func (r *ndjsonReporter) Error(err error) {
	r.writeLine(errorDoc(err, r.base.redact))
}

func (r *ndjsonReporter) Debug(msg string, attrs ...Attr) {
	if !r.base.opts.Debug {
		return
	}
	m := map[string]any{"v": 1, "type": "debug", "message": r.base.redact(msg)}
	if len(attrs) > 0 {
		am := map[string]string{}
		for _, a := range attrs {
			am[a.Key] = r.base.redact(a.Value)
		}
		m["attrs"] = am
	}
	r.writeLine(m)
}

func (r *ndjsonReporter) writeLine(v any) {
	r.base.mu.Lock()
	defer r.base.mu.Unlock()
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(r.base.opts.Out, "{\"v\":1,\"type\":\"error\",\"code\":\"ERR_M_INTERNAL\",\"message\":\"marshal\",\"exit\":1}\n")
		return
	}
	b = append(b, '\n')
	_, _ = r.base.opts.Out.Write(b)
}

type jsonReporter struct{ base *baseReporter }

func (r *jsonReporter) Progress(Event) {}

func (r *jsonReporter) Error(err error) {
	r.base.mu.Lock()
	defer r.base.mu.Unlock()
	doc := errorDoc(err, r.base.redact)
	b, err2 := json.MarshalIndent(doc, "", "  ")
	if err2 != nil {
		fmt.Fprintln(r.base.opts.Out, `{"v":1,"type":"error","code":"ERR_M_INTERNAL","message":"marshal","exit":1}`)
		return
	}
	fmt.Fprintln(r.base.opts.Out, string(b))
}

func (r *jsonReporter) Debug(msg string, attrs ...Attr) {
	if !r.base.opts.Debug {
		return
	}
	r.base.mu.Lock()
	defer r.base.mu.Unlock()
	fmt.Fprintf(r.base.opts.Err, "debug: %s\n", r.base.redact(msg))
}

type silentReporter struct{ base *baseReporter }

func (r *silentReporter) Progress(Event)        {}
func (r *silentReporter) Debug(string, ...Attr) {}
func (r *silentReporter) Error(err error) {
	r.base.mu.Lock()
	defer r.base.mu.Unlock()
	fmt.Fprintln(r.base.opts.Err, r.base.redact(formatHumanError(err)))
}

func formatHumanError(err error) string {
	if err == nil {
		return ""
	}
	var ae *apperr.Error
	if errors.As(err, &ae) && ae != nil {
		return ae.Error()
	}
	return err.Error()
}

func errorDoc(err error, redact func(string) string) ErrorDocument {
	doc := ErrorDocument{V: 1, Type: "error", Code: string(apperr.Internal), Message: "unknown", Exit: 1}
	if err == nil {
		doc.Code = string(apperr.OK)
		doc.Message = ""
		doc.Exit = 0
		return doc
	}
	var ae *apperr.Error
	if errors.As(err, &ae) && ae != nil {
		doc.Code = string(ae.Code)
		doc.Op = redact(ae.Op)
		doc.Subject = redact(ae.Subject)
		doc.Message = redact(ae.Message)
		if doc.Message == "" && ae.Cause != nil {
			doc.Message = redact(ae.Cause.Error())
		}
		doc.Exit = apperr.ExitCode(ae)
		return doc
	}
	doc.Message = redact(err.Error())
	doc.Exit = apperr.ExitCode(err)
	doc.Code = string(apperr.CodeOf(err))
	return doc
}

func truncate(s string, width int) string {
	if width <= 0 || utf8.RuneCountInString(s) <= width {
		return s
	}
	runes := []rune(s)
	if width < 4 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

var (
	reBearer    = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*`)
	reQueryTok  = regexp.MustCompile(`(?i)([?&](?:access_token|authToken|token|password|secret)=)([^&\s]+)`)
	reEnvSecret = regexp.MustCompile(`(?i)([A-Z0-9_]*(?:TOKEN|PASSWORD|SECRET|API_KEY)=)([^\s&]+)`)
)

// Redact removes credentials and common secret shapes from s (fail-closed).
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = redactURLs(s)
	s = reBearer.ReplaceAllString(s, "Bearer ***")
	s = reQueryTok.ReplaceAllString(s, "${1}***")
	s = reEnvSecret.ReplaceAllString(s, "${1}***")
	return s
}

func redactURLs(s string) string {
	if u, err := url.Parse(s); err == nil && u.Scheme != "" && u.Host != "" && u.User != nil {
		u.User = url.UserPassword("***", "***")
		return u.String()
	}
	parts := strings.Fields(s)
	changed := false
	for i, p := range parts {
		u, err := url.Parse(p)
		if err != nil || u.Scheme == "" || u.Host == "" || u.User == nil {
			continue
		}
		u.User = url.UserPassword("***", "***")
		parts[i] = u.String()
		changed = true
	}
	if changed {
		return strings.Join(parts, " ")
	}
	return s
}

// FormatErrorDocument builds the JSON error document (exported for tests).
func FormatErrorDocument(err error, unsafe bool) ErrorDocument {
	redact := Redact
	if unsafe {
		redact = func(s string) string { return s }
	}
	return errorDoc(err, redact)
}
