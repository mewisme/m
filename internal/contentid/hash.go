package contentid

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// NewHasher returns a hash.Hash for a supported digest algorithm.
func NewHasher(algo string) (hash.Hash, error) {
	switch strings.ToLower(algo) {
	case "sha512":
		return sha512.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha1":
		return sha1.New(), nil
	default:
		return nil, apperr.New(apperr.Integrity, "contentid.hash", algo, "unsupported algorithm")
	}
}

// HexDigest returns the lowercase hex digest of data using algo.
func HexDigest(algo string, data []byte) (string, error) {
	h, err := NewHasher(algo)
	if err != nil {
		return "", err
	}
	if _, err := h.Write(data); err != nil {
		return "", apperr.Wrap(apperr.IO, "contentid.digest", algo, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// MatchHex reports whether data digest matches wantHex for algo.
func MatchHex(data []byte, algo, wantHex string) error {
	got, err := HexDigest(algo, data)
	if err != nil {
		return err
	}
	if got != wantHex {
		return apperr.New(apperr.Integrity, "contentid.digest", algo+"/"+wantHex,
			fmt.Sprintf("digest mismatch: got %s want %s", got, wantHex))
	}
	return nil
}

// RejectUnsafeKeyPath rejects store key paths with traversal or platform escapes.
func RejectUnsafeKeyPath(s string) error {
	if s == "" {
		return apperr.New(apperr.Store, "contentid.key", s, "empty key")
	}
	if strings.Contains(s, "..") {
		return apperr.New(apperr.Store, "contentid.key", s, "invalid key")
	}
	if len(s) >= 2 && s[1] == ':' {
		return apperr.New(apperr.Store, "contentid.key", s, "drive path")
	}
	if strings.HasPrefix(s, "//") {
		return apperr.New(apperr.Store, "contentid.key", s, "unc path")
	}
	if strings.Contains(s, "\\") {
		return apperr.New(apperr.Store, "contentid.key", s, "invalid key separators")
	}
	if strings.HasPrefix(s, "/") {
		return apperr.New(apperr.Store, "contentid.key", s, "absolute path")
	}
	return nil
}
