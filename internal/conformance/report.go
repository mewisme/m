package conformance

import (
	"encoding/json"
	"time"
)

// Report is the machine-readable outcome of a core certification run.
type Report struct {
	SchemaVersion int           `json:"schemaVersion"`
	Matrix        string        `json:"matrix"`
	StartedAt     time.Time     `json:"startedAt"`
	FinishedAt    time.Time     `json:"finishedAt"`
	Passed        bool          `json:"passed"`
	Filter        string        `json:"filter,omitempty"`
	DryRun        bool          `json:"dryRun,omitempty"`
	Suites        []SuiteResult `json:"suites"`
}

// SuiteResult records one suite execution.
type SuiteResult struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Package    string `json:"package"`
	Run        string `json:"run"`
	Required   bool   `json:"required"`
	Status     string `json:"status"`
	Duration   string `json:"duration,omitempty"`
	ExitCode   int    `json:"exitCode,omitempty"`
	Error      string `json:"error,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
	SkipReason string `json:"skipReason,omitempty"`
}

const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
	StatusPlanned = "planned"
)

// EncodeJSON returns indented JSON for report.
func (r Report) EncodeJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
