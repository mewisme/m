package provenance

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// TrustPolicy selects which public key material backs signature verification.
type TrustPolicy int

const (
	// TrustFixtureKey uses the embedded fixture ed25519 key (tests only).
	TrustFixtureKey TrustPolicy = iota + 1
	// TrustConfiguredKey uses VerifyOptions.TrustedPublicKey.
	TrustConfiguredKey
	// TrustSigstoreRoots verifies against Sigstore/Fulcio roots (not yet supported).
	TrustSigstoreRoots
)

const npmEcosystem = "npm"

// PackageBinding pins attestation subject matching to one package identity.
type PackageBinding struct {
	Name      string
	Version   string
	PURL      string
	Digest    TarballDigest
	Ecosystem string
}

// BindingFromNameVersion builds an npm package binding with optional digest.
func BindingFromNameVersion(name, version string, digest TarballDigest) PackageBinding {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	purl := ""
	if name != "" && version != "" {
		purl = "pkg:npm/" + name + "@" + version
	}
	return PackageBinding{
		Name:      name,
		Version:   version,
		PURL:      purl,
		Digest:    digest,
		Ecosystem: npmEcosystem,
	}
}

// ParsePublicKeyBase64 decodes a base64 ed25519 public key.
func ParsePublicKeyBase64(b64 string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("decode trusted public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("trusted public key must be %d bytes, got %d", ed25519.PublicKeySize, len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

func resolveTrust(opts VerifyOptions) (ed25519.PublicKey, string, error) {
	switch opts.TrustPolicy {
	case TrustFixtureKey:
		return FixturePublicKey(), "fixture", nil
	case TrustConfiguredKey:
		if opts.TrustedPublicKey == nil {
			return nil, "", fmt.Errorf("configured trust requires trusted public key")
		}
		return opts.TrustedPublicKey, "configured", nil
	case TrustSigstoreRoots:
		return nil, "", fmt.Errorf("sigstore root trust is not supported yet")
	default:
		return nil, "", fmt.Errorf("trust policy required")
	}
}
