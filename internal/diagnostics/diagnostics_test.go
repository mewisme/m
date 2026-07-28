package diagnostics_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/testkit"
)

func TestRedactCases(t *testing.T) {
	root := testkit.ModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "testdata", "diagnostics", "redact-cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		In   string `json:"in"`
		Want string `json:"want"`
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		got := diagnostics.Redact(c.In)
		if got != c.Want {
			t.Errorf("Redact(%q)=%q want %q", c.In, got, c.Want)
		}
	}
}

func TestJSONErrorGolden(t *testing.T) {
	ae := apperr.New(apperr.Lockfile, "read", "https://user:secret@registry.npmjs.org/pkg", "ambiguous lockfile")
	doc := diagnostics.FormatErrorDocument(ae, false)
	got, marshalErr := json.MarshalIndent(doc, "", "  ")
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	got = append(bytes.TrimSpace(got), '\n')
	root := testkit.ModuleRoot(t)
	want, readErr := os.ReadFile(filepath.Join(root, "testdata", "diagnostics", "error-golden.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(want) {
		t.Fatalf("golden mismatch:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
	if !strings.Contains(doc.Subject, "***") && !strings.Contains(doc.Subject, "%2A%2A%2A") {
		t.Fatal("subject must redact URL credentials")
	}
	if doc.V != 1 || doc.Type != "error" || doc.Code == "" || doc.Message == "" {
		t.Fatalf("invalid error doc: %+v", doc)
	}
}

func TestProgressGoldenNDJSON(t *testing.T) {
	var out bytes.Buffer
	r := diagnostics.NewReporter(diagnostics.Options{
		Out:    &out,
		Err:    ioDiscard{},
		Format: "ndjson",
		Color:  diagnostics.ColorNever,
	})
	total := int64(100)
	r.Progress(diagnostics.Event{
		V: 1, Type: "progress", Phase: "fetch", Package: "left-pad@1.0.0",
		Bytes: 50, TotalBytes: &total, OpID: "op-1", TxID: nil,
	})
	root := testkit.ModuleRoot(t)
	want, err := os.ReadFile(filepath.Join(root, "testdata", "diagnostics", "progress-golden.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != string(want) {
		t.Fatalf("progress golden mismatch:\n got %q\nwant %q", out.String(), string(want))
	}
}

func TestNDJSONLineAtomicConcurrent(t *testing.T) {
	var out bytes.Buffer
	r := diagnostics.NewReporter(diagnostics.Options{
		Out:    &out,
		Err:    ioDiscard{},
		Format: "ndjson",
		Color:  diagnostics.ColorNever,
	})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			r.Progress(diagnostics.Event{Phase: "fetch", Package: "p", Bytes: int64(n), OpID: "x"})
		}(i)
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 50 {
		t.Fatalf("want 50 lines, got %d", len(lines))
	}
	for _, line := range lines {
		var ev diagnostics.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("broken line %q: %v", line, err)
		}
	}
}

func TestSilentDropsProgress(t *testing.T) {
	var out, errW bytes.Buffer
	r := diagnostics.NewReporter(diagnostics.Options{Out: &out, Err: &errW, Format: "silent"})
	r.Progress(diagnostics.Event{Phase: "fetch"})
	if out.Len() != 0 || errW.Len() != 0 {
		t.Fatal("silent must drop progress")
	}
	r.Error(apperr.New(apperr.Usage, "cli", "", "bad flag"))
	if !strings.Contains(errW.String(), "ERR_M_USAGE") {
		t.Fatalf("silent must still print errors: %q", errW.String())
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
