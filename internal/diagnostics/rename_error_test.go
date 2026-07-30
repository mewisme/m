package diagnostics_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/fsx"
)

func TestHumanRenameErrorNotTruncatedAtNarrowWidth(t *testing.T) {
	src := strings.Repeat(`F:\very\long\path\`, 8) + `node_modules`
	dst := src + ".mew-old"
	re := fsx.NewRenameError("rename", src, dst, errors.New("Access is denied."))
	err := apperr.Wrap(apperr.IO, "transaction.publish", dst, re)
	var errBuf bytes.Buffer
	rep := diagnostics.NewReporter(diagnostics.Options{
		Err:       &errBuf,
		Format:    "default",
		Color:     diagnostics.ColorNever,
		TermWidth: 40,
	})
	rep.Error(err)
	out := errBuf.String()
	if strings.HasSuffix(strings.TrimSpace(out), "...") {
		t.Fatalf("fatal error truncated at narrow width: %q", out)
	}
	if !strings.Contains(out, "source:") || !strings.Contains(out, "destination:") {
		t.Fatalf("missing structured rename fields: %q", out)
	}
}

func TestJSONRenameErrorFields(t *testing.T) {
	re := fsx.NewRenameError("rename", `F:\src\node_modules`, `F:\dst\node_modules`, errors.New("sharing violation"))
	err := apperr.Wrap(apperr.IO, "transaction.publish", `F:\dst\node_modules`, re)
	doc := diagnostics.FormatErrorDocument(err, false)
	if doc.Source == "" || doc.Destination == "" || doc.Cause == "" {
		t.Fatalf("missing rename fields: %+v", doc)
	}
	if strings.HasSuffix(doc.Message, "...") {
		t.Fatalf("truncated json message: %q", doc.Message)
	}
}
