package testkit

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// DiffReportSchemaVersion versions differential comparison reports.
const DiffReportSchemaVersion = 1

// ToolRun records one tool invocation for differential comparison.
type ToolRun struct {
	Name     string   `json:"name"`
	Args     []string `json:"args,omitempty"`
	ExitCode int      `json:"exitCode"`
	Stdout   string   `json:"stdout,omitempty"`
	Stderr   string   `json:"stderr,omitempty"`
}

// DiffItem is one normalized difference between Mew and a reference tool.
type DiffItem struct {
	Field  string `json:"field"`
	Mew    string `json:"mew,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// DiffReport is the conformance differential report schema (0080 consumes this).
type DiffReport struct {
	SchemaVersion int        `json:"schemaVersion"`
	Skipped       bool       `json:"skipped,omitempty"`
	SkipReason    string     `json:"skipReason,omitempty"`
	Mew           ToolRun    `json:"mew"`
	Reference     ToolRun    `json:"reference"`
	Diffs         []DiffItem `json:"diffs"`
}

var (
	reAbsUnix = regexp.MustCompile(`(?i)(/?(?:Users|home|tmp|var|private)/[^\s"']+)`)
	reAbsWin  = regexp.MustCompile(`(?i)([A-Z]:\\[^\s"']+)`)
	reISOTime = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?`)
)

// NormalizeOutput strips volatile absolute paths, CRLF, and timestamps for comparison.
func NormalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = reAbsWin.ReplaceAllString(s, "<ABS>")
	s = reAbsUnix.ReplaceAllString(s, "<ABS>")
	s = reISOTime.ReplaceAllString(s, "<TIME>")
	return strings.TrimSpace(s)
}

// WriteDiffReport encodes a report with indent and trailing newline.
func WriteDiffReport(path string, r *DiffReport) error {
	if r == nil {
		r = &DiffReport{}
	}
	if r.SchemaVersion == 0 {
		r.SchemaVersion = DiffReportSchemaVersion
	}
	if r.Diffs == nil {
		r.Diffs = []DiffItem{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// ReadDiffReport loads and validates a differential report.
func ReadDiffReport(path string) (*DiffReport, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r DiffReport
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.SchemaVersion == 0 {
		r.SchemaVersion = DiffReportSchemaVersion
	}
	if r.Diffs == nil {
		r.Diffs = []DiffItem{}
	}
	return &r, nil
}
