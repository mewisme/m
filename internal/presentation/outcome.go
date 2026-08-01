package presentation

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatDuration renders milliseconds as a human-readable duration string.
// Examples: "842ms", "1.24s", "12.4s", "2m 14s".
func FormatDuration(ms int64) string {
	if ms < 1 {
		ms = 1
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	sec := float64(ms) / 1000
	if sec < 10 {
		// Up to 2 fractional digits, trim trailing zeros.
		s := fmt.Sprintf("%.2f", sec)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		return s + "s"
	}
	if sec < 60 {
		// 1 fractional digit.
		s := fmt.Sprintf("%.1f", sec)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		return s + "s"
	}
	// >= 60 seconds: "2m 14s".
	m := int64(sec) / 60
	s := int64(sec) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

// FormatMutationCompletion builds the compact install-family completion line.
// Examples:
//
//	54 packages installed [4.36s]
//	54 packages installed, 3 removed [4.36s]
//	3 packages removed [842ms]
//	1 package installed [842ms]
//	Already up to date [1.24s]
func FormatMutationCompletion(added, updated, removed int, durationMs int64) string {
	installed := added + updated
	var b strings.Builder
	if installed > 0 {
		if installed == 1 {
			b.WriteString("1 package installed")
		} else {
			fmt.Fprintf(&b, "%d packages installed", installed)
		}
	}
	if removed > 0 {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		if removed == 1 {
			b.WriteString("1 package removed")
		} else {
			fmt.Fprintf(&b, "%d packages removed", removed)
		}
	}
	if b.Len() == 0 {
		b.WriteString("Already up to date")
	}
	if durationMs > 0 {
		fmt.Fprintf(&b, " [%s]", FormatDuration(durationMs))
	}
	return b.String()
}

// FormatCounts builds a compact human-readable count summary.
// Zero-valued groups are omitted. Example: "24 reused, 8 downloaded".
func FormatCounts(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		val := pairs[i]
		label := pairs[i+1]
		if val == "" || val == "0" {
			continue
		}
		parts = append(parts, val+" "+label)
	}
	return strings.Join(parts, ", ")
}

// FormatCountsInt builds a compact count summary from int values and labels.
func FormatCountsInt(pairs ...any) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		var valStr string
		switch v := pairs[i].(type) {
		case int:
			if v == 0 {
				continue
			}
			valStr = strconv.Itoa(v)
		case int64:
			if v == 0 {
				continue
			}
			valStr = strconv.FormatInt(v, 10)
		default:
			continue
		}
		label, ok := pairs[i+1].(string)
		if !ok {
			continue
		}
		parts = append(parts, valStr+" "+label)
	}
	return strings.Join(parts, ", ")
}

// InstallCounts returns formatted reused/downloaded/linked counts for install summaries.
func InstallCounts(reused, downloaded, linked int) string {
	return FormatCountsInt(reused, "reused", downloaded, "downloaded", linked, "linked")
}

// FormatInvocationHeader builds the command invocation header line.
// Example: "mew install v1.2.3 (a1b2c3d)" or "m install dev (a1b2c3d)".
// version and commit are optional; the header always includes binary and command path.
func FormatInvocationHeader(binary, commandPath, version, commit string) string {
	var b strings.Builder
	b.WriteString(binary)
	if commandPath != "" {
		b.WriteByte(' ')
		b.WriteString(commandPath)
	}
	if version != "" {
		b.WriteByte(' ')
		b.WriteString(version)
	} else {
		b.WriteString(" dev")
	}
	if commit != "" {
		short := commit
		if len(short) > 7 {
			short = short[:7]
		}
		b.WriteString(" (")
		b.WriteString(short)
		b.WriteByte(')')
	}
	return b.String()
}

// ShortCommit returns a 7-character commit prefix, or empty string.
func ShortCommit(commit string) string {
	if len(commit) < 7 {
		return commit
	}
	return commit[:7]
}
