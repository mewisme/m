package lifecycle

import (
	"testing"
)

func TestPrepareCacheHit(t *testing.T) {
	dir := t.TempDir()
	script := Script{
		PackageName: "counter",
		PackageKey:  "counter@1.0.0",
		Name:        "prepare",
		Integrity:   "sha256-deadbeef",
	}
	hit, err := cacheHit(dir, script)
	if err != nil || hit {
		t.Fatalf("initial hit=%v err=%v", hit, err)
	}
	if err := markCache(dir, script); err != nil {
		t.Fatal(err)
	}
	hit, err = cacheHit(dir, script)
	if err != nil || !hit {
		t.Fatalf("want cache hit, got hit=%v err=%v", hit, err)
	}
}
