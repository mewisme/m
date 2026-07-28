package manifest_test

import (
	"bytes"
	"testing"

	"github.com/mewisme/mew/internal/manifest"
)

func TestManifestNormalizeRoundTrip(t *testing.T) {
	raw := []byte(`{
  "schemaVersion": 1,
  "name": "app",
  "version": "1.0.0",
  "dependencies": [
    {"name": "ms", "range": "^2.0.0", "kind": "prod"},
    {"name": "left-pad", "range": "^1.0.0", "kind": "prod"}
  ]
}
`)
	m, err := manifest.ParseJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := manifest.EncodeJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	// Encode sorts deps: left-pad before ms
	again, err := manifest.ParseJSON(got)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manifest.EncodeJSON(again)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, second) {
		t.Fatalf("not stable\n%s\n%s", got, second)
	}
	if again.Dependencies[0].Name != "left-pad" {
		t.Fatalf("expected sorted deps, got %v", again.Dependencies)
	}
}
