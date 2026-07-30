package presentation

import "strings"

type plainRenderer struct {
	settings EffectiveSettings
}

func newPlainRenderer(settings EffectiveSettings) *plainRenderer {
	return &plainRenderer{settings: settings}
}

func (r *plainRenderer) Settings() EffectiveSettings { return r.settings }

func (r *plainRenderer) Status(line StatusLine) string {
	sym := statusSymbol(r.settings.Symbols, line.Status)
	var b strings.Builder
	if sym != "" {
		b.WriteString(sym)
		b.WriteByte(' ')
	}
	b.WriteString(line.Text)
	if line.Detail != "" {
		b.WriteByte(' ')
		b.WriteString(line.Detail)
	}
	return b.String()
}

func (r *plainRenderer) KeyValues(rows []KeyValue) string {
	return formatKeyValues(rows, r.settings, false, Theme{})
}

func (r *plainRenderer) Notice(n Notice) string {
	sym := statusSymbol(r.settings.Symbols, n.Status)
	if n.Status == StatusNone {
		sym = r.settings.Symbols.Warning
	}
	if sym == "" {
		return n.Message
	}
	return sym + " " + n.Message
}

func (r *plainRenderer) Hint(h Hint) string {
	arrow := r.settings.Symbols.Arrow
	if arrow == "" {
		return h.Message
	}
	return arrow + " " + h.Message
}

func (r *plainRenderer) Summary(s Summary) string {
	parts := []string{r.Status(StatusLine{Status: s.Status, Text: s.Title})}
	if len(s.Metrics) > 0 {
		parts = append(parts, "", r.KeyValues(s.Metrics))
	}
	for _, n := range s.Notices {
		parts = append(parts, r.Notice(n))
	}
	for _, h := range s.Hints {
		parts = append(parts, r.Hint(h))
	}
	return strings.Join(parts, "\n")
}

func (r *plainRenderer) PackageDeltas(deltas []PackageDelta) string {
	return formatPackageDeltas(deltas, r.settings, false, Theme{})
}

func (r *plainRenderer) Table(m TableModel) string {
	return formatTable(m, r.settings, false, Theme{})
}

func statusSymbol(s Symbols, st Status) string {
	switch st {
	case StatusSuccess:
		return s.Success
	case StatusWarning:
		return s.Warning
	case StatusError:
		return s.Error
	case StatusInfo:
		return s.Info
	default:
		return ""
	}
}

func formatKeyValues(rows []KeyValue, settings EffectiveSettings, color bool, theme Theme) string {
	if len(rows) == 0 {
		return ""
	}
	stacked := settings.Width < 60 || settings.Accessible
	if stacked {
		var b strings.Builder
		for i, row := range rows {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(applyStyle(theme.Label, row.Key, color))
			b.WriteString(": ")
			b.WriteString(styleValue(row.Value, row.Style, color, theme))
		}
		return b.String()
	}
	maxKey := 0
	for _, row := range rows {
		if w := CellWidth(row.Key); w > maxKey {
			maxKey = w
		}
	}
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("  ")
		b.WriteString(applyStyle(theme.Label, row.Key, color))
		pad := maxKey - CellWidth(row.Key)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad+2))
		b.WriteString(styleValue(row.Value, row.Style, color, theme))
	}
	return b.String()
}

func styleValue(val string, kind ValueKind, color bool, theme Theme) string {
	if !color {
		return val
	}
	switch kind {
	case ValuePackage:
		return applyStyle(theme.Package, val, true)
	case ValueVersion:
		return applyStyle(theme.Version, val, true)
	case ValuePath:
		return applyStyle(theme.Path, val, true)
	case ValueCommand:
		return applyStyle(theme.Command, val, true)
	case ValueNumber:
		return applyStyle(theme.Number, val, true)
	case ValueMuted:
		return applyStyle(theme.Muted, val, true)
	default:
		return applyStyle(theme.Value, val, true)
	}
}

func formatPackageDeltas(deltas []PackageDelta, settings EffectiveSettings, color bool, theme Theme) string {
	if len(deltas) == 0 {
		return ""
	}
	sym := settings.Symbols
	var b strings.Builder
	for i, d := range deltas {
		if i > 0 {
			b.WriteByte('\n')
		}
		var mark string
		switch d.Kind {
		case DeltaAdded:
			mark = sym.Added
			if color {
				mark = applyStyle(theme.Added, mark, true)
			}
		case DeltaRemoved:
			mark = sym.Removed
			if color {
				mark = applyStyle(theme.Removed, mark, true)
			}
		default:
			mark = "~"
			if color {
				mark = applyStyle(theme.Updated, mark, true)
			}
		}
		b.WriteString(mark)
		b.WriteByte(' ')
		name := d.Name
		if color {
			name = applyStyle(theme.Package, name, true)
		}
		b.WriteString(name)
		b.WriteString("  ")
		switch d.Kind {
		case DeltaUpdated:
			from, to := d.From, d.To
			if from == "" {
				from = d.Version
			}
			if to == "" {
				to = d.Version
			}
			if color {
				from = applyStyle(theme.Version, from, true)
				to = applyStyle(theme.Version, to, true)
			}
			b.WriteString(from)
			b.WriteByte(' ')
			b.WriteString(sym.Arrow)
			b.WriteByte(' ')
			b.WriteString(to)
		default:
			ver := d.Version
			if color {
				ver = applyStyle(theme.Version, ver, true)
			}
			b.WriteString(ver)
		}
	}
	return b.String()
}
