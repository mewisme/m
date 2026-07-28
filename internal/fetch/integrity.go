package fetch

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/url"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/contentid"
)

// ponytail: integrity parsing supports std SRI base64 and fixture hex digests; upgrade = strict SRI-only.

const maxBodyBytes = 512 << 20 // 512 MiB

// ParsedIntegrity is a normalized integrity expectation.
type ParsedIntegrity struct {
	Algo     string // sha512, sha256, sha1
	Hex      string // lowercase hex digest
	Original string // caller-supplied integrity string
}

// ParseIntegrity parses an npm dist.integrity SRI string (algo-base64 or algo-hex).
func ParseIntegrity(s string) (ParsedIntegrity, error) {
	s = strings.TrimSpace(s)
	id, err := contentid.ParseSRI(s)
	if err != nil {
		return ParsedIntegrity{}, err
	}
	return ParsedIntegrity{
		Algo:     id.Algo,
		Hex:      id.Hex,
		Original: s,
	}, nil
}

// ParseShasum parses a legacy npm dist.shasum sha1 hex string.
func ParseShasum(hexDigest string) (ParsedIntegrity, error) {
	hexDigest = strings.TrimSpace(strings.ToLower(hexDigest))
	if hexDigest == "" {
		return ParsedIntegrity{}, apperr.New(apperr.Integrity, "fetch.shasum", "", "empty shasum")
	}
	if err := contentid.ValidateKey("sha1", hexDigest); err != nil {
		return ParsedIntegrity{}, apperr.New(apperr.Integrity, "fetch.shasum", hexDigest, "invalid sha1 hex")
	}
	raw, err := hex.DecodeString(hexDigest)
	if err != nil || len(raw) != sha1.Size {
		return ParsedIntegrity{}, apperr.New(apperr.Integrity, "fetch.shasum", hexDigest, "invalid sha1 hex")
	}
	return ParsedIntegrity{
		Algo:     "sha1",
		Hex:      hexDigest,
		Original: "sha1-" + base64.StdEncoding.EncodeToString(raw),
	}, nil
}

// ExpectedIntegrity resolves integrity from SRI and optional legacy shasum.
// SRI wins when both are present.
func ExpectedIntegrity(integrity, shasum string) (ParsedIntegrity, error) {
	if strings.TrimSpace(integrity) != "" {
		return ParseIntegrity(integrity)
	}
	if strings.TrimSpace(shasum) != "" {
		return ParseShasum(shasum)
	}
	return ParsedIntegrity{}, apperr.New(apperr.Integrity, "fetch.integrity", "", "missing integrity and shasum")
}

// VerifyReader hashes r while reading and compares to expected integrity or shasum.
func VerifyReader(r io.Reader, integrity, shasum string) (parsed ParsedIntegrity, n int64, err error) {
	expected, err := ExpectedIntegrity(integrity, shasum)
	if err != nil {
		return ParsedIntegrity{}, 0, err
	}
	h, err := newHasher(expected.Algo)
	if err != nil {
		return ParsedIntegrity{}, 0, err
	}
	limited := &io.LimitedReader{R: r, N: maxBodyBytes + 1}
	written, err := io.Copy(h, limited)
	if err != nil {
		return ParsedIntegrity{}, written, apperr.Wrap(apperr.IO, "fetch.verify", expected.Algo, err)
	}
	if written > maxBodyBytes {
		return ParsedIntegrity{}, written, apperr.New(apperr.Integrity, "fetch.verify", expected.Algo,
			fmt.Sprintf("body exceeds %d bytes", maxBodyBytes))
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected.Hex {
		subj := expected.Original
		if subj == "" {
			subj = expected.Algo
		}
		return ParsedIntegrity{}, written, apperr.New(apperr.Integrity, "fetch.verify", redactURL(subj),
			fmt.Sprintf("digest mismatch: got %s want %s", got, expected.Hex))
	}
	return expected, written, nil
}

// BlobPath returns <algo>/<hex> for content-addressed blob storage.
func (p ParsedIntegrity) BlobPath() string {
	return p.Algo + "/" + p.Hex
}

// Identity returns the normalized store key identity.
func (p ParsedIntegrity) Identity() contentid.Identity {
	return contentid.Identity{Algo: p.Algo, Hex: p.Hex}
}

// RedactURL removes query strings from URLs in error subjects.
func RedactURL(raw string) string {
	if raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		if i := strings.IndexByte(raw, '?'); i >= 0 {
			return raw[:i] + "?…"
		}
		return raw
	}
	if u.RawQuery != "" {
		u.RawQuery = ""
		u.Fragment = ""
		return u.String() + "?…"
	}
	return u.String()
}

func redactURL(s string) string { return RedactURL(s) }

func newHasher(algo string) (hash.Hash, error) {
	switch algo {
	case "sha512":
		return sha512.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha1":
		return sha1.New(), nil
	default:
		return nil, apperr.New(apperr.Integrity, "fetch.hash", algo, "unsupported algorithm")
	}
}
