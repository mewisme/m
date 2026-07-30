package envexec

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashHex returns lowercase SHA-256 hex for arbitrary input.
func HashHex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
