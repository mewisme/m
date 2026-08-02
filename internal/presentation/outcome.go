package presentation

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatDuration renders a time.Duration as a human-readable string.
// Rules:
//   - <1ms: integer nanoseconds, e.g. "820ns"
//   - <1s: integer milliseconds, e.g. "317ms"
//   - <1m: seconds with ≤2 decimal places, e.g. "4.36s"
//   - ≥1m: minutes and seconds, e.g. "1m4.36s"
//
// Never outputs microseconds. Never outputs 0ms for a non-zero duration.
// Trailing zeros are trimmed.
func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "0ms"
	}
	ns := d.Nanoseconds()
	// Sub-millisecond: integer nanoseconds.
	if ns < 1_000_000 {
		return fmt.Sprintf("%dns", ns)
	}
	ms := ns / 1_000_000
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	sec := float64(ns) / 1_000_000_000.0
	if sec < 60 {
		s := fmt.Sprintf("%.2f", sec)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		return s + "s"
	}
	// ≥ 60 seconds: "1m4.36s", "12m8s".
	m := int64(sec) / 60
	rem := sec - float64(m*60)
	if rem < 0.005 {
		return fmt.Sprintf("%dm", m)
	}
	s := fmt.Sprintf("%.2f", rem)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return fmt.Sprintf("%dm%ss", m, s)
}

// FormatDurationMs is a convenience wrapper for millisecond int64 values.
func FormatDurationMs(ms int64) string {
	return FormatDuration(time.Duration(ms) * time.Millisecond)
}

// FormatMutationCompletion builds the compact install-family completion message
// (without duration). Examples:
//
//	54 packages installed
//	54 packages installed, 3 removed
//	3 packages removed
//	1 package installed
//	Already up to date
func FormatMutationCompletion(added, updated, removed int) string {
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

// BuildInfo carries version metadata for the invocation header.
type BuildInfo struct {
	Version     string
	ShortCommit string
	Dirty       bool
}

// FormatInvocationHeader builds the command invocation header line.
// Example: "mew install v1.2.3 (a1b2c3d)" or "m install dev+dirty (a1b2c3d)".
// binary is the invoked binary name (m, mew, mx, mewx).
// commandPath is the relative command path after the binary (e.g., "install", "add lodash").
func FormatInvocationHeader(binary, commandPath string, info BuildInfo) string {
	var b strings.Builder
	b.WriteString(binary)
	if commandPath != "" {
		b.WriteByte(' ')
		b.WriteString(commandPath)
	}
	if info.Version != "" {
		b.WriteByte(' ')
		b.WriteString(info.Version)
	} else {
		b.WriteString(" dev")
	}
	if info.Dirty {
		b.WriteString("+dirty")
	}
	if info.ShortCommit != "" {
		b.WriteString(" (")
		b.WriteString(info.ShortCommit)
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
