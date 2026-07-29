package pnpm

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
)

// NormalizePatchBytes normalizes patch file content for hashing (CRLF→LF).
func NormalizePatchBytes(data []byte) []byte {
	return []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
}

// ComputePatchHash returns the patch identity for a supported pnpm producer major.
// pnpm 9 uses MD5 with RFC 4648 base32 (lowercase, no padding).
// pnpm 10 and 11 use SHA-256 hex of normalized bytes.
func ComputePatchHash(producerMajor int, normalized []byte) (string, error) {
	switch producerMajor {
	case 9:
		return patchHashMD5Base32(normalized), nil
	case 10, 11:
		sum := sha256.Sum256(normalized)
		return hex.EncodeToString(sum[:]), nil
	default:
		return "", fmt.Errorf("unsupported pnpm producer major %d for patch hash", producerMajor)
	}
}

func patchHashMD5Base32(data []byte) string {
	sum := md5.Sum(data)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return strings.ToLower(enc.EncodeToString(sum[:]))
}
