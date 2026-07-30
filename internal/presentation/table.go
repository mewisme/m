package presentation

import (
	"sort"
	"strings"
)

const stackedTableThreshold = 60

func formatTable(m TableModel, settings EffectiveSettings, color bool, theme Theme) string {
	if len(m.Columns) == 0 {
		return ""
	}
	rows := append([]map[string]string(nil), m.Rows...)
	sort.SliceStable(rows, func(i, j int) bool {
		a := rowSortKey(rows[i], m.Columns)
		b := rowSortKey(rows[j], m.Columns)
		return a < b
	})

	if settings.Width < stackedTableThreshold || settings.Accessible {
		return formatStackedTable(m.Columns, rows, settings, color, theme)
	}
	return formatWideTable(m.Columns, rows, settings, color, theme)
}

func rowSortKey(row map[string]string, cols []TableColumn) string {
	var parts []string
	for _, c := range cols {
		parts = append(parts, row[c.Key])
	}
	return strings.Join(parts, "\x00")
}

func formatStackedTable(cols []TableColumn, rows []map[string]string, settings EffectiveSettings, color bool, theme Theme) string {
	primary := cols[0]
	for _, c := range cols {
		if c.Primary {
			primary = c
			break
		}
	}
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
			b.WriteByte('\n')
		}
		title := row[primary.Key]
		b.WriteString(applyStyle(theme.Header, title, color))
		for _, c := range cols {
			if c.Key == primary.Key {
				continue
			}
			b.WriteByte('\n')
			b.WriteString("  ")
			b.WriteString(applyStyle(theme.Label, strings.ToLower(c.Header), color))
			b.WriteString("  ")
			b.WriteString(styleValue(row[c.Key], ValuePlain, color, theme))
		}
	}
	_ = settings
	return b.String()
}

func formatWideTable(cols []TableColumn, rows []map[string]string, settings EffectiveSettings, color bool, theme Theme) string {
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = CellWidth(c.Header)
		if c.MinWidth > widths[i] {
			widths[i] = c.MinWidth
		}
	}
	for _, row := range rows {
		for i, c := range cols {
			w := CellWidth(row[c.Key])
			if w > widths[i] {
				widths[i] = w
			}
		}
	}
	for i, c := range cols {
		if c.Prefer > 0 && widths[i] > c.Prefer {
			widths[i] = c.Prefer
		}
	}
	widths = fitTableWidths(widths, cols, settings.Width)

	var b strings.Builder
	for i, c := range cols {
		if i > 0 {
			b.WriteString("  ")
		}
		h := applyStyle(theme.Header, c.Header, color)
		b.WriteString(padCell(h, c.Header, widths[i], c.Align))
	}
	b.WriteByte('\n')
	for ri, row := range rows {
		if ri > 0 {
			b.WriteByte('\n')
		}
		for i, c := range cols {
			if i > 0 {
				b.WriteString("  ")
			}
			raw := row[c.Key]
			cell := fitCell(raw, widths[i], c.Truncate, settings.Symbols.Ellipsis)
			styled := styleValue(cell, ValuePlain, color, theme)
			b.WriteString(padCell(styled, cell, widths[i], c.Align))
		}
	}
	return b.String()
}

func fitTableWidths(widths []int, cols []TableColumn, termWidth int) []int {
	sep := 2 * (len(widths) - 1)
	if sep < 0 {
		sep = 0
	}
	total := sep
	for _, w := range widths {
		total += w
	}
	if total <= termWidth || termWidth <= 0 {
		return widths
	}
	overflow := total - termWidth
	out := append([]int(nil), widths...)
	for overflow > 0 {
		idx := -1
		best := 0
		for i, w := range out {
			min := cols[i].MinWidth
			if min <= 0 {
				min = 4
			}
			slack := w - min
			if slack > best {
				best = slack
				idx = i
			}
		}
		if idx < 0 || best <= 0 {
			break
		}
		out[idx]--
		overflow--
	}
	return out
}

func fitCell(s string, width int, policy TruncatePolicy, ellipsis string) string {
	if width <= 0 {
		return ""
	}
	if CellWidth(s) <= width {
		return s
	}
	switch policy {
	case TruncateMiddle:
		return MiddleTruncate(s, width, ellipsis)
	case TruncateWrap:
		lines := WrapWords(s, width)
		if len(lines) == 0 {
			return ""
		}
		return lines[0]
	default:
		return MiddleTruncate(s, width, ellipsis)
	}
}

func padCell(styled, plain string, width int, align ColumnAlign) string {
	pad := width - CellWidth(plain)
	if pad < 0 {
		pad = 0
	}
	if align == AlignRight {
		return strings.Repeat(" ", pad) + styled
	}
	return styled + strings.Repeat(" ", pad)
}
