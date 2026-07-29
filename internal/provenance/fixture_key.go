package provenance

import (
	"crypto/ed25519"
	_ "embed"
	"encoding/base64"
)

//go:embed testdata/fixture-ed25519.pub
var fixturePublicKeyB64 string

// FixturePublicKey is the trusted ed25519 public key for fixture attestations.
func FixturePublicKey() ed25519.PublicKey {
	raw, err := base64.StdEncoding.DecodeString(fixturePublicKeyB64)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		panic("provenance: invalid fixture public key")
	}
	return ed25519.PublicKey(raw)
}
