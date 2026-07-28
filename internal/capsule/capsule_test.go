package capsule_test

import (
	"bytes"
	"testing"

	"github.com/mewisme/mew/internal/capsule"
)

func TestCapsuleSortsPackages(t *testing.T) {
	c := &capsule.Capsule{
		SchemaVersion: capsule.SchemaVersion,
		ID:            "cap-1",
		Packages:      []string{"z@1.0.0", "a@1.0.0"},
		Integrity:     "sha512-x",
	}
	first, err := capsule.EncodeJSON(c)
	if err != nil {
		t.Fatal(err)
	}
	again, err := capsule.DecodeJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	if again.Packages[0] != "a@1.0.0" {
		t.Fatalf("unsorted: %v", again.Packages)
	}
	second, err := capsule.EncodeJSON(again)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("unstable")
	}
}
