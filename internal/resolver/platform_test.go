package resolver_test

import (
	"testing"

	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

func TestPlatformMatches(t *testing.T) {
	target := resolver.Target{OS: "linux", CPU: "x64", Libc: "glibc"}

	meta := registry.VersionMeta{}
	if !target.Matches(meta) {
		t.Fatal("empty constraints should match")
	}

	meta = registry.VersionMeta{OS: []string{"linux", "darwin"}}
	if !target.Matches(meta) {
		t.Fatal("expected linux match")
	}

	meta = registry.VersionMeta{OS: []string{"darwin"}}
	if target.Matches(meta) {
		t.Fatal("expected linux mismatch")
	}

	meta = registry.VersionMeta{CPU: []string{"arm64"}}
	if target.Matches(meta) {
		t.Fatal("expected cpu mismatch")
	}

	meta = registry.VersionMeta{Libc: []string{"musl"}}
	if target.Matches(meta) {
		t.Fatal("expected libc mismatch")
	}

	meta = registry.VersionMeta{Libc: []string{"glibc"}}
	if !target.Matches(meta) {
		t.Fatal("expected libc match")
	}
}

func TestCurrentTargetNormalization(t *testing.T) {
	tgt := resolver.CurrentTarget()
	if tgt.OS == "" || tgt.CPU == "" {
		t.Fatalf("target=%+v", tgt)
	}
}

func TestNormalizeCPU(t *testing.T) {
	// Indirect via CurrentTarget on amd64 hosts; explicit check via Matches with x64 list.
	target := resolver.Target{OS: "linux", CPU: "x64", Libc: ""}
	meta := registry.VersionMeta{CPU: []string{"x64"}}
	if !target.Matches(meta) {
		t.Fatal("expected x64 match")
	}
}
