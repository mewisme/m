package diagnostics_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
)

func TestNoticeNDJSON(t *testing.T) {
	var out bytes.Buffer
	rep := diagnostics.NewReporter(diagnostics.Options{
		Format: "ndjson",
		Out:    &out,
		Err:    &out,
	})
	rep.Notice(diagnostics.NoticeEvent{
		V:        1,
		Severity: "warn",
		Message:  "example",
		Hint:     "try again",
	})
	var doc map[string]any
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["type"] != "notice" {
		t.Fatalf("%v", doc)
	}
}

func TestOperationEventsHumanNoPanic(t *testing.T) {
	var errb bytes.Buffer
	rep := diagnostics.NewReporter(diagnostics.Options{Err: &errb})
	rep.OperationStarted(diagnostics.OperationStartedEvent{Kind: "install", Label: "root"})
	rep.OperationCompleted(diagnostics.OperationCompletedEvent{ID: "op-1", Status: "ok", DurationMs: 10})
	rep.Notice(diagnostics.NoticeEvent{Severity: "info", Message: "done"})
}
