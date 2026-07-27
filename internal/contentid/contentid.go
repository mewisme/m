package contentid

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/mewisme/m/internal/apperr"
)

// Identity is a normalized content address: lowercase algo and hex digest.
type Identity struct {
	Algo string
	Hex  string
}

// SRI returns the npm integrity form algo-hex.
func (id Identity) SRI() string {
	return id.Algo + "-" + id.Hex
}

// KeyPath returns algo/hex for store layout.
func (id Identity) KeyPath() string {
	return id.Algo + "/" + id.Hex
}

// ParseSRI parses an npm dist.integrity SRI string (algo-base64 or algo-hex).
func ParseSRI(s string) (Identity, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Identity{}, apperr.New(apperr.Integrity, "contentid.parse", "", "empty integrity")
	}
	algo, digest, ok := strings.Cut(s, "-")
	if !ok {
		return Identity{}, apperr.New(apperr.Integrity, "contentid.parse", s, "missing algorithm prefix")
	}
	algo = strings.ToLower(algo)
	raw, err := decodeDigest(algo, digest)
	if err != nil {
		return Identity{}, apperr.Wrap(apperr.Integrity, "contentid.parse", s, err)
	}
	id := Identity{Algo: algo, Hex: hex.EncodeToString(raw)}
	if err := ValidateKey(id.Algo, id.Hex); err != nil {
		return Identity{}, err
	}
	return id, nil
}

// ValidateKey checks algo and lowercase hex digest for store keys.
func ValidateKey(algo, hexDigest string) error {
	algo = strings.ToLower(strings.TrimSpace(algo))
	hexDigest = strings.TrimSpace(hexDigest)
	if algo == "" || hexDigest == "" {
		return apperr.New(apperr.Store, "contentid.key", algo+"/"+hexDigest, "invalid key")
	}
	if strings.ContainsAny(algo, "-/\\.") || strings.Contains(hexDigest, "/") {
		return apperr.New(apperr.Store, "contentid.key", algo+"/"+hexDigest, "invalid key separators")
	}
	switch algo {
	case "sha512", "sha256", "sha1":
	default:
		return apperr.New(apperr.Store, "contentid.key", algo, "unsupported algorithm")
	}
	wantLen := digestHexLen(algo)
	if len(hexDigest) != wantLen {
		return apperr.New(apperr.Store, "contentid.key", algo+"/"+hexDigest,
			fmt.Sprintf("digest length %d want %d", len(hexDigest), wantLen))
	}
	for _, c := range hexDigest {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return apperr.New(apperr.Store, "contentid.key", algo+"/"+hexDigest, "digest must be lowercase hex")
		}
	}
	return nil
}

func digestHexLen(algo string) int {
	switch algo {
	case "sha512":
		return sha512.Size * 2
	case "sha256":
		return sha256.Size * 2
	case "sha1":
		return sha1.Size * 2
	default:
		return 0
	}
}

func decodeDigest(algo, digest string) ([]byte, error) {
	switch algo {
	case "sha512", "sha256", "sha1":
	default:
		return nil, fmt.Errorf("unsupported algorithm %q", algo)
	}
	if isHexDigest(digest) {
		raw, err := hex.DecodeString(strings.ToLower(digest))
		if err != nil {
			return nil, err
		}
		if err := checkDigestLen(algo, raw); err != nil {
			return nil, err
		}
		return raw, nil
	}
	raw, err := base64.StdEncoding.DecodeString(digest)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(digest)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 digest")
		}
	}
	if err := checkDigestLen(algo, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func isHexDigest(s string) bool {
	if len(s)%2 != 0 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func checkDigestLen(algo string, raw []byte) error {
	want := 0
	switch algo {
	case "sha512":
		want = sha512.Size
	case "sha256":
		want = sha256.Size
	case "sha1":
		want = sha1.Size
	}
	if len(raw) != want {
		return fmt.Errorf("digest length %d want %d for %s", len(raw), want, algo)
	}
	return nil
}
