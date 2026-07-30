package cli

import (
	"testing"
)

func TestMXReservedNamesDrift(t *testing.T) {
	root := NewMXRoot(testBuildInfo())
	want := []string{"version", "completion", "cache"}
	for _, name := range want {
		if !IsMXReserved(root, name) {
			t.Fatalf("expected reserved %q", name)
		}
	}
	if IsMXReserved(root, "prettier") {
		t.Fatal("prettier should not be reserved")
	}
}

func TestIsMXReservedCache(t *testing.T) {
	root := NewMXRoot(testBuildInfo())
	if !IsMXReserved(root, "cache") {
		t.Fatal("cache should be reserved")
	}
	if IsMXReserved(root, "prettier") {
		t.Fatal("prettier should not be reserved")
	}
}
