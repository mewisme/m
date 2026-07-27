package resolver_test

import (
	"testing"

	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

func TestPlatformMatrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		target resolver.Target
		meta   registry.VersionMeta
		want   bool
	}{
		{
			name:   "empty matches all",
			target: resolver.Target{OS: "linux", CPU: "x64"},
			meta:   registry.VersionMeta{},
			want:   true,
		},
		{
			name:   "positive os match",
			target: resolver.Target{OS: "linux", CPU: "x64"},
			meta:   registry.VersionMeta{OS: []string{"linux", "darwin"}},
			want:   true,
		},
		{
			name:   "positive os miss",
			target: resolver.Target{OS: "linux", CPU: "x64"},
			meta:   registry.VersionMeta{OS: []string{"darwin"}},
			want:   false,
		},
		{
			name:   "negative os excluded",
			target: resolver.Target{OS: "win32", CPU: "x64"},
			meta:   registry.VersionMeta{OS: []string{"!win32"}},
			want:   false,
		},
		{
			name:   "negative os allowed",
			target: resolver.Target{OS: "linux", CPU: "x64"},
			meta:   registry.VersionMeta{OS: []string{"!win32"}},
			want:   true,
		},
		{
			name:   "mixed positive and negative",
			target: resolver.Target{OS: "linux", CPU: "x64"},
			meta:   registry.VersionMeta{OS: []string{"linux", "!darwin"}},
			want:   true,
		},
		{
			name:   "mixed excludes darwin",
			target: resolver.Target{OS: "darwin", CPU: "x64"},
			meta:   registry.VersionMeta{OS: []string{"linux", "!darwin"}},
			want:   false,
		},
		{
			name:   "cpu negative",
			target: resolver.Target{OS: "linux", CPU: "arm64"},
			meta:   registry.VersionMeta{CPU: []string{"!arm64"}},
			want:   false,
		},
		{
			name:   "libc negative",
			target: resolver.Target{OS: "linux", CPU: "x64", Libc: "musl"},
			meta:   registry.VersionMeta{Libc: []string{"!musl"}},
			want:   false,
		},
		{
			name:   "libc positive",
			target: resolver.Target{OS: "linux", CPU: "x64", Libc: "glibc"},
			meta:   registry.VersionMeta{Libc: []string{"glibc"}},
			want:   true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.target.Matches(tc.meta)
			if got != tc.want {
				t.Fatalf("Matches()=%v want %v target=%+v meta=%+v", got, tc.want, tc.target, tc.meta)
			}
		})
	}
}
