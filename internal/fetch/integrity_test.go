package fetch_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fetch"
)

func TestParseIntegrityHexFixture(t *testing.T) {
	p, err := fetch.ParseIntegrity("sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63")
	if err != nil {
		t.Fatal(err)
	}
	if p.Algo != "sha256" || len(p.Hex) != 64 {
		t.Fatalf("%+v", p)
	}
}

func TestParseIntegrityBase64(t *testing.T) {
	// sha256 of empty string
	p, err := fetch.ParseIntegrity("sha256-47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=")
	if err != nil {
		t.Fatal(err)
	}
	if p.Hex != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("hex=%s", p.Hex)
	}
}

func TestParseShasum(t *testing.T) {
	p, err := fetch.ParseShasum("758b80171fc185274170cb6db31a08042813d860")
	if err != nil {
		t.Fatal(err)
	}
	if p.Algo != "sha1" {
		t.Fatalf("%+v", p)
	}
}

func TestVerifyReaderMatchAndMismatch(t *testing.T) {
	body := []byte("hello tarball")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	_, _, err := fetch.VerifyReader(bytes.NewReader(body), integrity, "")
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("want integrity error, got %v", err)
	}
}

func TestRedactURL(t *testing.T) {
	got := fetch.RedactURL("https://registry.example/pkg.tgz?token=secret&sig=abc")
	if strings.Contains(got, "secret") {
		t.Fatalf("leaked token: %q", got)
	}
	if !strings.Contains(got, "?") {
		t.Fatalf("missing redaction marker: %q", got)
	}
}
