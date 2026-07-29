package provenance

import (
	"crypto/ed25519"
	_ "embed"
)

//go:embed testdata/fixture-ed25519.pub
var fixturePublicKeyB64 string

// FixturePublicKey is the trusted ed25519 public key for fixture attestations.
func FixturePublicKey() ed25519.PublicKey {
	pub, err := ParsePublicKeyBase64(fixturePublicKeyB64)
	if err != nil {
		panic("provenance: invalid fixture public key: " + err.Error())
	}
	return pub
}

// FixturePublicKeyBase64 returns the embedded fixture public key (base64).
func FixturePublicKeyBase64() string {
	return fixturePublicKeyB64
}
