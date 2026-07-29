package resolver_test

import (
	"encoding/json"
	"testing"

	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/resolver"
)

func TestNormalizeGitURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://github.com/foo/bar.git", "https://github.com/foo/bar.git"},
		{"git@github.com:foo/bar.git", "ssh://git@github.com/foo/bar.git"},
		{"//github.com/foo/bar.git", "https://github.com/foo/bar.git"},
	}
	for _, tc := range cases {
		got, err := resolver.NormalizeGitURL(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q => %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidateGitURLRejectsUnsafe(t *testing.T) {
	if err := resolver.ValidateGitURL("ftp://example.com/a.git"); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
	if err := resolver.ValidateGitURL("https:///missing-host.git"); err == nil {
		t.Fatal("expected missing host error")
	}
}

func TestParseGitRange(t *testing.T) {
	parsed, err := resolver.ParseGitRange("https://github.com/foo/bar.git#deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.URL != "https://github.com/foo/bar.git" || parsed.Ref != "deadbeef" {
		t.Fatalf("got %+v", parsed)
	}
}

func TestGitExtensionRoundTrip(t *testing.T) {
	raw, err := json.Marshal(map[string]resolver.GitSource{
		"pkg@abc1234": {URL: "https://github.com/foo/bar.git", Commit: "abc1234"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ext := lockfile.Extensions{resolver.GitExtensionKey: raw}
	got, err := resolver.DecodeGitSources(ext)
	if err != nil {
		t.Fatal(err)
	}
	if src, ok := got["pkg@abc1234"]; !ok || src.Commit != "abc1234" {
		t.Fatalf("got %#v", got)
	}
}

func TestValidateGitURLAllowsFile(t *testing.T) {
	if err := resolver.ValidateGitURL("file:///tmp/repo.git"); err != nil {
		t.Fatal(err)
	}
}
