package cli

import (
	"testing"

	"github.com/mewisme/mew/internal/apperr"
)

func TestParseMXInvocationModeA(t *testing.T) {
	inv, err := ParseMXInvocation([]string{"typescript", "--", "--version"})
	if err != nil {
		t.Fatal(err)
	}
	if !inv.ModeA || len(inv.PackageSpecs) != 1 || inv.PackageSpecs[0].Name != "typescript" {
		t.Fatalf("%+v", inv)
	}
}

func TestParseMXInvocationModeB(t *testing.T) {
	inv, err := ParseMXInvocation([]string{"-p", "typescript", "tsc", "--version"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.ModeA || inv.Command != "tsc" {
		t.Fatalf("%+v", inv)
	}
}

func TestParseMXInvocationUnknownFlag(t *testing.T) {
	_, err := ParseMXInvocation([]string{"--nope"})
	if err == nil || apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("err=%v", err)
	}
}

func TestMXReservedDrift(t *testing.T) {
	root := NewMXRoot(BuildInfo{})
	names := MXReservedNames(root)
	want := map[string]bool{"version": true, "completion": true, "cache": true}
	for _, n := range names {
		delete(want, n)
	}
	if len(want) != 0 {
		t.Fatalf("missing reserved: %v", want)
	}
}
