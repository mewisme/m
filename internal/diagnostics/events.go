package diagnostics

// Metric is a typed numeric measurement on operation completion.
type Metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

// OperationStartedEvent marks the beginning of a logical operation.
type OperationStartedEvent struct {
	V     int    `json:"v"`
	Type  string `json:"type"`
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Total *int64 `json:"total,omitempty"`
	Unit  string `json:"unit,omitempty"`
}

// OperationProgressEvent reports incremental operation progress.
type OperationProgressEvent struct {
	V         int    `json:"v"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	Completed int64  `json:"completed"`
	Total     *int64 `json:"total,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// OperationCompletedEvent marks operation completion.
type OperationCompletedEvent struct {
	V          int      `json:"v"`
	Type       string   `json:"type"`
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	DurationMs int64    `json:"duration_ms"`
	Metrics    []Metric `json:"metrics,omitempty"`
}

// NoticeEvent is a typed user-facing notice independent of terminal styling.
type NoticeEvent struct {
	V        int    `json:"v"`
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}
