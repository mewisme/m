package advisory

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mewisme/mew/internal/semver"
)

// FailOnLevel is the minimum advisory severity that causes audit to exit nonzero.
type FailOnLevel string

const (
	FailOnNone     FailOnLevel = "none"
	FailOnLow      FailOnLevel = "low"
	FailOnModerate FailOnLevel = "moderate"
	FailOnHigh     FailOnLevel = "high"
	FailOnCritical FailOnLevel = "critical"
)

// DBWarning records a non-fatal issue in a loaded advisory database entry.
type DBWarning struct {
	EntryID string `json:"entryId"`
	Message string `json:"message"`
}

type semverInterval struct {
	introduced   string
	fixed        string
	lastAffected string
}

// ParseFailOn parses the audit --fail-on flag value.
func ParseFailOn(s string) (FailOnLevel, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return FailOnNone, nil
	}
	switch FailOnLevel(s) {
	case FailOnNone, FailOnLow, FailOnModerate, FailOnHigh, FailOnCritical:
		return FailOnLevel(s), nil
	default:
		return "", fmt.Errorf("invalid --fail-on %q (want none|low|moderate|high|critical)", s)
	}
}

// SeverityRank maps a vulnerability severity label to a comparable rank.
// Unknown labels and empty values are treated as low for deterministic CI gates.
func SeverityRank(severity string) int {
	s := strings.TrimSpace(strings.ToLower(severity))
	switch s {
	case "", "unknown":
		return 1
	case "low":
		return 1
	case "moderate", "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	default:
		if score, err := strconv.ParseFloat(s, 64); err == nil {
			return cvssToRank(score)
		}
		return 1
	}
}

func failOnMinRank(level FailOnLevel) int {
	switch level {
	case FailOnNone:
		return 0
	case FailOnLow:
		return 1
	case FailOnModerate:
		return 2
	case FailOnHigh:
		return 3
	case FailOnCritical:
		return 4
	default:
		return 0
	}
}

// ReportExceedsThreshold reports whether report contains a finding at or above level.
func ReportExceedsThreshold(report AuditReport, level FailOnLevel) bool {
	minRank := failOnMinRank(level)
	if minRank == 0 || len(report.Vulnerabilities) == 0 {
		return false
	}
	for _, v := range report.Vulnerabilities {
		if SeverityRank(v.Severity) >= minRank {
			return true
		}
	}
	return false
}

func cvssToRank(score float64) int {
	switch {
	case score >= 9.0:
		return 4
	case score >= 7.0:
		return 3
	case score >= 4.0:
		return 2
	default:
		return 1
	}
}

func normalizeAuditVersion(version string) string {
	if i := strings.IndexByte(version, '#'); i >= 0 {
		return version[:i]
	}
	return version
}

func versionMatchesRange(r VersionRange, version string) bool {
	if r.Type != "" && r.Type != "SEMVER" {
		return false
	}
	version = normalizeAuditVersion(version)
	intervals, _ := buildSemverIntervals(r.Events)
	if len(intervals) == 0 {
		return false
	}
	for _, iv := range intervals {
		if versionInInterval(version, iv) {
			return true
		}
	}
	return false
}

func buildSemverIntervals(events []RangeEvent) ([]semverInterval, []string) {
	var intervals []semverInterval
	var warnings []string
	var current *semverInterval

	for _, ev := range events {
		hasIntro := ev.Introduced != ""
		hasFixed := ev.Fixed != ""
		hasLast := ev.LastAffected != ""
		fields := boolToInt(hasIntro) + boolToInt(hasFixed) + boolToInt(hasLast)
		if fields == 0 {
			warnings = append(warnings, "empty range event")
			continue
		}
		if fields > 1 {
			warnings = append(warnings, "range event has multiple fields")
			continue
		}
		switch {
		case hasIntro:
			if current != nil {
				intervals = append(intervals, *current)
			}
			current = &semverInterval{introduced: ev.Introduced}
		case hasFixed:
			if current == nil {
				warnings = append(warnings, "fixed without introduced")
				continue
			}
			current.fixed = ev.Fixed
			intervals = append(intervals, *current)
			current = nil
		case hasLast:
			if current == nil {
				warnings = append(warnings, "last_affected without introduced")
				continue
			}
			current.lastAffected = ev.LastAffected
			intervals = append(intervals, *current)
			current = nil
		}
	}
	if current != nil {
		intervals = append(intervals, *current)
	}
	if len(events) > 0 && len(intervals) == 0 && len(warnings) == 0 {
		warnings = append(warnings, "range produced no intervals")
	}
	return intervals, warnings
}

func versionInInterval(version string, iv semverInterval) bool {
	if iv.introduced == "" {
		return false
	}
	if iv.introduced != "0" {
		cmp, err := semver.Compare(version, iv.introduced)
		if err != nil || cmp < 0 {
			return false
		}
	}
	if iv.fixed != "" {
		cmp, err := semver.Compare(version, iv.fixed)
		if err != nil || cmp >= 0 {
			return false
		}
		return true
	}
	if iv.lastAffected != "" {
		cmp, err := semver.Compare(version, iv.lastAffected)
		if err != nil || cmp > 0 {
			return false
		}
		return true
	}
	return iv.introduced != "0" || true
}

func collectRangeWarnings(entries []OSVEntry) []DBWarning {
	var out []DBWarning
	for _, entry := range entries {
		for _, aff := range entry.Affected {
			for _, r := range aff.Ranges {
				if r.Type != "" && r.Type != "SEMVER" {
					continue
				}
				_, ws := buildSemverIntervals(r.Events)
				for _, msg := range ws {
					out = append(out, DBWarning{EntryID: entry.ID, Message: msg})
				}
			}
		}
	}
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
