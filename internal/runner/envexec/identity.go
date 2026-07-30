package envexec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"runtime"
	"strings"
)

// IdentitySchemaVersion is the current EnvironmentIdentity JSON schema version.
const IdentitySchemaVersion = 1

var digestHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// BareDigestHex strips an optional sha256: prefix for event and inspect emission.
func BareDigestHex(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "sha256:")
}

// EnvironmentIdentity is the typed security identity for a materialized environment.
// Command names and child args are intentionally excluded.
type EnvironmentIdentity struct {
	SchemaVersion  int                 `json:"schemaVersion"`
	Source         SourceKind          `json:"source"`
	GraphDigest    string              `json:"graphDigest"`
	MaterialDigest string              `json:"materialDigest"`
	SourceDigest   string              `json:"sourceDigest"`
	Platform       PlatformFingerprint `json:"platform"`
	LinkerMode     string              `json:"linkerMode"`
	NodeABI        string              `json:"nodeAbi,omitempty"`
}

// PlatformFingerprint captures the execution platform binding.
type PlatformFingerprint struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// CurrentPlatform returns the host platform fingerprint.
func CurrentPlatform() PlatformFingerprint {
	return PlatformFingerprint{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// IdentityDigest returns a stable hex digest of the canonical identity JSON.
func (id EnvironmentIdentity) IdentityDigest() string {
	return digestCanonical(id)
}

func digestCanonical(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		b = []byte{}
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
