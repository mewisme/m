package contentid_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/contentid"
)

func TestHexDigestAndMatchHex(t *testing.T) {
	data := []byte("mew")
	got, err := contentid.HexDigest("sha256", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 64 {
		t.Fatalf("len=%d", len(got))
	}
	if err := contentid.MatchHex(data, "sha256", got); err != nil {
		t.Fatal(err)
	}
	if err := contentid.MatchHex(data, "sha256", strings.Repeat("a", 64)); err == nil {
		t.Fatal("expected digest mismatch")
	}
}

func TestRejectUnsafeKeyPath(t *testing.T) {
	cases := []string{"", "../sha256/abc", "sha256/../abc", "C:sha256/abc", "//sha256/abc", `sha256\abc`, "/sha256/abc"}
	for _, key := range cases {
		if err := contentid.RejectUnsafeKeyPath(key); err == nil {
			t.Fatalf("RejectUnsafeKeyPath(%q) expected error", key)
		}
	}
	if err := contentid.RejectUnsafeKeyPath("sha256/" + strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
}
