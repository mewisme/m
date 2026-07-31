package node

import (
	"context"
	"os/exec"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{"22.11.0", "22.11.0", false},
		{"20.18.1", "20.18.1", false},
		{"16.20.2", "16.20.2", false},
		{"12.22.12", "12.22.12", false},
		{"18.0", "18.0.0", false},
		{"22", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got, err := normalizeVersion(tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Errorf("normalizeVersion(%q) expected error", tt.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeVersion(%q) unexpected error: %v", tt.raw, err)
			continue
		}
		if got != tt.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestDetectCapabilities(t *testing.T) {
	tests := []struct {
		version string
		want    []string
	}{
		{"22.11.0", []string{"import-preload", "require-preload"}},
		{"16.0.0", []string{"import-preload", "require-preload"}},
		{"14.0.0", []string{"require-preload"}},
		{"12.0.0", []string{"require-preload"}},
		{"10.0.0", nil},
	}
	for _, tt := range tests {
		got := detectCapabilities(tt.version)
		if !strSliceEq(got, tt.want) {
			t.Errorf("detectCapabilities(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestDiscoverNodeFound(t *testing.T) {
	// Only run if node is available on PATH.
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	inst, err := Discover(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if inst.ExePath == "" {
		t.Fatal("empty ExePath")
	}
	if inst.NormalizedVersion == "" {
		t.Fatal("empty NormalizedVersion")
	}
	if inst.DiscoverySource != "PATH" {
		t.Fatalf("DiscoverySource = %q, want PATH", inst.DiscoverySource)
	}
}

func TestDiscoverNodeNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := Discover(ctx, Request{ExplicitCandidate: "no-such-node-exe-999"})
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

func strSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
