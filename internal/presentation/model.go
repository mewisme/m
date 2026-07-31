package presentation

// Status is a semantic status for status lines and summaries.
type Status int

const (
	StatusNone Status = iota
	StatusSuccess
	StatusWarning
	StatusError
	StatusInfo
)

// ValueKind selects value styling for key-value rows.
type ValueKind int

const (
	ValuePlain ValueKind = iota
	ValuePackage
	ValueVersion
	ValuePath
	ValueCommand
	ValueNumber
	ValueMuted
)

// StatusLine is a one-line command outcome.
type StatusLine struct {
	Status Status
	Text   string
	Detail string
}

// KeyValue is one labeled field.
type KeyValue struct {
	Key   string
	Value string
	Style ValueKind
}

// Notice is a non-fatal advisory.
type Notice struct {
	Status  Status
	Message string
}

// Hint is a follow-up action suggestion.
type Hint struct {
	Message string
}

// Summary is a command outcome block.
type Summary struct {
	Status  Status
	Title   string
	Deltas  []PackageDelta
	Metrics []KeyValue
	Notices []Notice
	Hints   []Hint
}

// DeltaKind classifies a package change row.
type DeltaKind int

const (
	DeltaAdded DeltaKind = iota
	DeltaUpdated
	DeltaRemoved
)

// PackageDelta is one package mutation row.
type PackageDelta struct {
	Kind    DeltaKind
	Name    string
	Version string
	From    string
	To      string
}

// PackageDeltaOptions controls how PackageDelta lists are rendered.
type PackageDeltaOptions struct {
	GroupByKind bool
	MaxRows     int // 0 means unlimited.
}

// maxSummaryPackageDeltas bounds human delta lists to prevent unbounded output.
const maxSummaryPackageDeltas = 50

// ColumnAlign is table cell alignment.
type ColumnAlign int

const (
	AlignLeft ColumnAlign = iota
	AlignRight
)

// TruncatePolicy controls overflow for a column.
type TruncatePolicy int

const (
	TruncateEnd TruncatePolicy = iota
	TruncateMiddle
	TruncateWrap
)

// TableColumn defines one table column.
type TableColumn struct {
	Key      string
	Header   string
	MinWidth int
	Prefer   int
	Align    ColumnAlign
	Truncate TruncatePolicy
	Primary  bool // first column used as stacked title
}

// TableModel is a borderless table.
type TableModel struct {
	Columns []TableColumn
	Rows    []map[string]string
}
