package presentation

// CauseView is one debug-only error cause row.
type CauseView struct {
	Label   string
	Message string
}

// ErrorView is the human error presentation model.
type ErrorView struct {
	Severity  Status
	Title     string
	Message   string
	Code      string
	Operation string
	Subject   string
	Context   []KeyValue
	Hints     []Hint
	Causes    []CauseView
	DocsTopic string
}
