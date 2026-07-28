package contentid_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/contentid"
)

func TestParseSRIRealSHA512Base64(t *testing.T) {
	// npm lodash 4.17.21 tarball integrity (sha512 with base64 including / + =)
	const sri = "sha512-v2hAJgGqb1MSsRazXu/8+S1mMBuy+AN7l/2n4svS9/WuTj9eeG1kbmRnkb80FrJ/dS/ZV+0ow45X9B0nQ3DT5Q=="
	id, err := contentid.ParseSRI(sri)
	if err != nil {
		t.Fatal(err)
	}
	if id.Algo != "sha512" || len(id.Hex) != 128 {
		t.Fatalf("%+v", id)
	}
	if strings.ContainsAny(id.Hex, "/+=") {
		t.Fatalf("hex must not contain base64 chars: %q", id.Hex)
	}
}

func TestParseSRISHA256Base64SlashPlusPadding(t *testing.T) {
	id, err := contentid.ParseSRI("sha256-47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=")
	if err != nil {
		t.Fatal(err)
	}
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if id.Hex != want {
		t.Fatalf("hex=%s want %s", id.Hex, want)
	}
}

func TestParseSRIHexFixture(t *testing.T) {
	id, err := contentid.ParseSRI("sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63")
	if err != nil {
		t.Fatal(err)
	}
	if id.Algo != "sha256" || len(id.Hex) != 64 {
		t.Fatalf("%+v", id)
	}
}

func TestValidateKeyRejectsInvalid(t *testing.T) {
	cases := []struct {
		algo, hex string
	}{
		{"sha256", "ABC"},
		{"sha256", strings.Repeat("a", 63)},
		{"md5", strings.Repeat("a", 32)},
		{"sha256", "aa/bb"},
		{"sha-256", strings.Repeat("a", 64)},
	}
	for _, tc := range cases {
		if err := contentid.ValidateKey(tc.algo, tc.hex); err == nil {
			t.Fatalf("expected error for %q/%q", tc.algo, tc.hex)
		}
	}
}

func TestValidateKeyAcceptsSHA1Hex(t *testing.T) {
	hex := strings.Repeat("a", 40)
	if err := contentid.ValidateKey("sha1", hex); err != nil {
		t.Fatal(err)
	}
}
