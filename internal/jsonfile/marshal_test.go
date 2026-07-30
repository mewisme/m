package jsonfile

import (
	"bytes"
	"testing"
)

func TestMarshalIndentAndTrailingNewline(t *testing.T) {
	raw, err := Marshal(map[string]any{"a": 1, "b": []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(raw, []byte("\n")) {
		t.Fatalf("missing trailing newline: %q", raw)
	}
	body := bytes.TrimSuffix(raw, []byte("\n"))
	want := "{\n  \"a\": 1,\n  \"b\": [\n    \"x\"\n  ]\n}"
	if string(body) != want {
		t.Fatalf("body=%q want=%q", body, want)
	}
}
