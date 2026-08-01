package presentation

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatDuration renders milliseconds as a human-readable duration string.
// Examples: "1ms", "180ms", "6.8s", "12s".
func FormatDuration(ms int64) string {
	if ms < 1 {
		ms = 1
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	sec := float64(ms) / 1000
	if sec < 10 {
		return fmt.Sprintf("%.1fs", sec)
	}
	return fmt.Sprintf("%.0fs", sec)
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
