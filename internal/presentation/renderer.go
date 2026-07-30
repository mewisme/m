package presentation

// StaticRenderer turns presentation models into deterministic strings.
// Implementations must not read environment, write I/O, or use clocks.
type StaticRenderer interface {
	Status(StatusLine) string
	KeyValues([]KeyValue) string
	Summary(Summary) string
	Notice(Notice) string
	Hint(Hint) string
	Error(ErrorView) string
	Table(TableModel) string
	PackageDeltas([]PackageDelta) string
	PlainText(string) string
	Settings() EffectiveSettings
}

// NewStaticRenderer returns a plain or rich renderer from settings.
// Legacy and ThemeNone always use the plain renderer (zero ANSI).
func NewStaticRenderer(settings EffectiveSettings) StaticRenderer {
	if settings.Legacy || settings.ThemeMode == ThemeNone || !settings.UseColor {
		return newPlainRenderer(settings)
	}
	return newRichRenderer(settings)
}
