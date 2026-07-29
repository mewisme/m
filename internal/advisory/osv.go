package advisory

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

const ReportSchemaVersion = 1

// OSVEntry is one OSV-compatible advisory record.
type OSVEntry struct {
	ID       string     `json:"id"`
	Aliases  []string   `json:"aliases,omitempty"`
	Summary  string     `json:"summary,omitempty"`
	Details  string     `json:"details,omitempty"`
	Affected []Affected `json:"affected"`
	Severity []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity,omitempty"`
	DatabaseSpecific map[string]any `json:"database_specific,omitempty"`
}

// Affected describes vulnerable package versions.
type Affected struct {
	Package struct {
		Ecosystem string `json:"ecosystem"`
		Name      string `json:"name"`
	} `json:"package"`
	Versions []string       `json:"versions,omitempty"`
	Ranges   []VersionRange `json:"ranges,omitempty"`
}

// VersionRange is a semver event range from OSV.
type VersionRange struct {
	Type   string       `json:"type"`
	Events []RangeEvent `json:"events"`
}

// RangeEvent is one introduced/fixed/last_affected boundary.
type RangeEvent struct {
	Introduced   string `json:"introduced,omitempty"`
	Fixed        string `json:"fixed,omitempty"`
	LastAffected string `json:"last_affected,omitempty"`
}

// AdvisoryDB is a loaded OSV database.
type AdvisoryDB struct {
	Entries  []OSVEntry
	Digest   string
	Warnings []DBWarning
}

// Vulnerability is one matched finding in an audit report.
type Vulnerability struct {
	ID       string `json:"id"`
	Package  string `json:"package"`
	Version  string `json:"version"`
	Severity string `json:"severity,omitempty"`
	Title    string `json:"title"`
	URL      string `json:"url,omitempty"`
}

// AuditReport is the JSON schema v1 audit output.
type AuditReport struct {
	SchemaVersion   int             `json:"schemaVersion"`
	ScannedAt       string          `json:"scannedAt"`
	DBDigest        string          `json:"dbDigest"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

// FixSuggestion recommends a non-vulnerable version bump (suggest only).
type FixSuggestion struct {
	Package       string `json:"package"`
	FromVersion   string `json:"fromVersion"`
	ToVersion     string `json:"toVersion"`
	Vulnerability string `json:"vulnerability"`
}

// Load parses OSV-compatible JSON (array of entries) into an AdvisoryDB.
func Load(data []byte) (*AdvisoryDB, error) {
	var entries []OSVEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, apperr.Wrap(apperr.Integrity, "advisory.load", "osv", err)
	}
	digest := Digest(data)
	db := &AdvisoryDB{Entries: entries, Digest: digest, Warnings: collectRangeWarnings(entries)}
	return db, nil
}

// MatchGraph returns sorted findings for packages in g.
func (db *AdvisoryDB) MatchGraph(g *graph.Graph) AuditReport {
	report := AuditReport{
		SchemaVersion:   ReportSchemaVersion,
		ScannedAt:       time.Now().UTC().Format(time.RFC3339),
		Vulnerabilities: nil,
	}
	if db != nil {
		report.DBDigest = db.Digest
	}
	if db == nil || g == nil {
		return report
	}
	seen := make(map[string]struct{})
	for _, pkg := range g.Packages {
		name := pkg.ID.Name
		version := pkg.ID.Version
		if name == "" || version == "" {
			continue
		}
		for _, entry := range db.Entries {
			if !db.entryMatches(entry, name, version) {
				continue
			}
			v := Vulnerability{
				ID:       displayID(entry),
				Package:  name,
				Version:  version,
				Severity: entrySeverity(entry),
				Title:    entryTitle(entry),
				URL:      entryURL(entry),
			}
			key := v.ID + "\x00" + v.Package + "\x00" + v.Version
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			report.Vulnerabilities = append(report.Vulnerabilities, v)
		}
	}
	sort.Slice(report.Vulnerabilities, func(i, j int) bool {
		a, b := report.Vulnerabilities[i], report.Vulnerabilities[j]
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.ID < b.ID
	})
	return report
}

// IsVulnerable reports whether name@version matches the database.
func (db *AdvisoryDB) IsVulnerable(name, version string) bool {
	if db == nil {
		return false
	}
	for _, entry := range db.Entries {
		if db.entryMatches(entry, name, version) {
			return true
		}
	}
	return false
}

func (db *AdvisoryDB) entryMatches(entry OSVEntry, name, version string) bool {
	version = normalizeAuditVersion(version)
	for _, aff := range entry.Affected {
		if aff.Package.Ecosystem != "" && aff.Package.Ecosystem != "npm" {
			continue
		}
		if aff.Package.Name != name {
			continue
		}
		for _, v := range aff.Versions {
			if v == version {
				return true
			}
		}
		for _, r := range aff.Ranges {
			if versionMatchesRange(r, version) {
				return true
			}
		}
	}
	return false
}

func displayID(entry OSVEntry) string {
	for _, alias := range entry.Aliases {
		if strings.HasPrefix(alias, "CVE-") {
			return alias
		}
	}
	if entry.ID != "" {
		return entry.ID
	}
	return "UNKNOWN"
}

func entryTitle(entry OSVEntry) string {
	if entry.Summary != "" {
		return entry.Summary
	}
	if entry.Details != "" {
		return entry.Details
	}
	return displayID(entry)
}

func entrySeverity(entry OSVEntry) string {
	if entry.DatabaseSpecific != nil {
		if s, ok := entry.DatabaseSpecific["severity"].(string); ok && s != "" {
			return s
		}
	}
	if len(entry.Severity) > 0 && entry.Severity[0].Score != "" {
		return entry.Severity[0].Score
	}
	return ""
}

func entryURL(entry OSVEntry) string {
	if entry.ID != "" {
		return "https://osv.dev/vulnerability/" + entry.ID
	}
	return ""
}

// FormatTable renders a human-readable audit table.
func FormatTable(report AuditReport) string {
	if len(report.Vulnerabilities) == 0 {
		return "found 0 vulnerabilities"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "found %d vulnerabilities\n", len(report.Vulnerabilities))
	for _, v := range report.Vulnerabilities {
		sev := v.Severity
		if sev == "" {
			sev = "unknown"
		}
		fmt.Fprintf(&b, "%s  %s@%s  %s  %s\n", sev, v.Package, v.Version, v.ID, v.Title)
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatFixSuggestions renders suggested version bumps.
func FormatFixSuggestions(fixes []FixSuggestion) string {
	if len(fixes) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range fixes {
		fmt.Fprintf(&b, "suggest %s: %s -> %s (%s)\n", f.Package, f.FromVersion, f.ToVersion, f.Vulnerability)
	}
	return strings.TrimRight(b.String(), "\n")
}
