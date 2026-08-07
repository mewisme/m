package cli

import (
	"reflect"
	"testing"
)

func TestMatchPathPattern(t *testing.T) {
	tests := []struct {
		name      string
		specifier string
		pattern   string
		want      []string
	}{
		{
			name:      "exact match",
			specifier: "@app/core",
			pattern:   "@app/core",
			want:      []string{""},
		},
		{
			name:      "exact no match",
			specifier: "@app/other",
			pattern:   "@app/core",
			want:      nil,
		},
		{
			name:      "prefix wildcard",
			specifier: "@app/helpers",
			pattern:   "@app/*",
			want:      []string{"helpers"},
		},
		{
			name:      "prefix wildcard no match",
			specifier: "@other/helpers",
			pattern:   "@app/*",
			want:      nil,
		},
		{
			name:      "middle wildcard",
			specifier: "@app/helpers/utils",
			pattern:   "@app/*/utils",
			want:      []string{"helpers"},
		},
		{
			name:      "catch-all wildcard",
			specifier: "@scope/pkg",
			pattern:   "*",
			want:      []string{"@scope/pkg"},
		},
		{
			name:      "empty captures",
			specifier: "@app/",
			pattern:   "@app/*",
			want:      []string{""},
		},
		{
			name:      "no wildcard in pattern",
			specifier: "@app/helpers",
			pattern:   "@app/helpers",
			want:      []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPathPattern(tt.specifier, tt.pattern)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("matchPathPattern(%q, %q) = %v, want %v", tt.specifier, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchPathPatternOverlappingPrefixSuffix(t *testing.T) {
	// Pattern "ab*bc" with specifier "abc": prefix "ab" matches, suffix "bc" matches,
	// but specifier is too short (len 3 < 2+2). Must reject, not panic.
	got := matchPathPattern("abc", "ab*bc")
	if got != nil {
		t.Errorf("matchPathPattern(%q, %q) = %v, want nil (overlapping prefix/suffix)", "abc", "ab*bc", got)
	}
}

func TestMatchPathPatternMultiWildcard(t *testing.T) {
	// Multiple wildcards with sequential capture.
	got := matchPathPattern("@scope/pkg/sub/file", "@*/*/*")
	if got == nil {
		t.Fatal("expected match")
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 captures, got %d: %v", len(got), got)
	}
	if got[0] != "scope" || got[1] != "pkg" || got[2] != "sub/file" {
		t.Errorf("captures: %v", got)
	}
}
