package lifecycle

import (
	"testing"
)

func TestPrepareCacheMarkerDiagnosticOnly(t *testing.T) {
	dir := t.TempDir()
	script := Script{
		PackageName: "counter",
		PackageKey:  "counter@1.0.0",
		Name:        "prepare",
		Integrity:   "sha256-deadbeef",
	}
	if err := markCache(dir, script); err != nil {
		t.Fatal(err)
	}
	if err := MarkCacheForTest(dir, script); err != nil {
		t.Fatal(err)
	}
}
