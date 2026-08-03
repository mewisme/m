package config

import "os"

// This file holds the multi-file view of validation. The rules themselves live
// in validate.go; here they are only aggregated, so a caller that validates one
// file and a caller that validates a whole scope chain share one engine and one
// report shape.

// ValidationReport is the outcome of validating every file a scope selects.
//
// Scope is what the user asked for, which may be effective and therefore span
// more than one file. Files carries one entry per file that was actually read,
// in resolution order (user before project) so a report reads the way the
// layers apply.
type ValidationReport struct {
	Scope Scope              `json:"scope,omitempty"`
	Valid bool               `json:"valid"`
	Files []ValidationResult `json:"files"`
}

// KeyCount is the number of leaf keys across every validated file.
func (r ValidationReport) KeyCount() int {
	n := 0
	for _, f := range r.Files {
		n += f.KeyCount
	}
	return n
}

// Diagnostics returns every finding across every file, already ordered: the
// per-file order is deterministic and files are visited in resolution order, so
// two runs over the same inputs produce identical reports.
func (r ValidationReport) Diagnostics() []Diagnostic {
	var out []Diagnostic
	for _, f := range r.Files {
		out = append(out, f.Diagnostics...)
	}
	return out
}

// Errors returns the error-severity diagnostics across every file.
func (r ValidationReport) Errors() []Diagnostic { return r.filter(SeverityError) }

// Warnings returns the warning-severity diagnostics across every file.
func (r ValidationReport) Warnings() []Diagnostic { return r.filter(SeverityWarning) }

func (r ValidationReport) filter(s Severity) []Diagnostic {
	var out []Diagnostic
	for _, d := range r.Diagnostics() {
		if d.Severity == s {
			out = append(out, d)
		}
	}
	return out
}

// ValidateFiles validates each file in order and aggregates the results.
//
// Every file is validated even when an earlier one fails, so a single broken
// file cannot hide problems in the rest. Scopes carries the per-file scope for
// the writable-scope check; a short or nil slice leaves the remaining files
// unscoped rather than guessing.
//
// A missing file contributes no result at all: an absent config is a legal
// state, and listing it would imply it was read.
func ValidateFiles(scope Scope, paths []string, scopes []Scope, opts ValidateOptions) ValidationReport {
	rep := ValidationReport{Scope: scope, Valid: true}
	for i, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil && os.IsNotExist(err) {
			// Absent optional config is valid; listing it would claim it was read.
			continue
		}
		fileOpts := opts
		if i < len(scopes) {
			fileOpts.Scope = scopes[i]
		} else {
			fileOpts.Scope = ""
		}
		res := ValidateFile(p, fileOpts)
		if !res.Valid() {
			rep.Valid = false
		}
		rep.Files = append(rep.Files, res)
	}
	return rep
}
