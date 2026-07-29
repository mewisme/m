package provenance

import (
	"encoding/json"
	"os"

	"github.com/mewisme/mew/internal/apperr"
)

const (
	mediaTypeBundleV03  = "application/vnd.dev.sigstore.bundle.v0.3+json"
	payloadTypeInToto   = "application/vnd.in-toto+json"
	statementTypeV1     = "https://in-toto.io/Statement/v1"
	predicateSLSAV1     = "https://slsa.dev/provenance/v1"
	predicateNPMPublish = "https://github.com/npm/attestation/tree/main/specs/publish/v0.1"
)

// Bundle is a Sigstore bundle subset used by npm provenance attestations.
type Bundle struct {
	MediaType            string               `json:"mediaType"`
	VerificationMaterial VerificationMaterial `json:"verificationMaterial"`
	DSSEEnvelope         DSSEEnvelope         `json:"dsseEnvelope"`
}

// VerificationMaterial holds public key bytes for fixture verification.
type VerificationMaterial struct {
	PublicKey *PublicKey `json:"publicKey,omitempty"`
}

// PublicKey is raw key material (base64) from a Sigstore bundle.
type PublicKey struct {
	RawBytes string `json:"rawBytes"`
}

// DSSEEnvelope is the signed attestation payload wrapper.
type DSSEEnvelope struct {
	PayloadType string    `json:"payloadType"`
	Payload     string    `json:"payload"`
	Signatures  []DSSESig `json:"signatures"`
}

// DSSESig is one DSSE signature entry.
type DSSESig struct {
	Sig   string `json:"sig"`
	KeyID string `json:"keyid,omitempty"`
}

// npmAttestationsFile is the registry/npm wrapper around one or more bundles.
type npmAttestationsFile struct {
	Attestations []npmAttestation `json:"attestations"`
}

type npmAttestation struct {
	PredicateType string `json:"predicateType"`
	Bundle        Bundle `json:"bundle"`
}

// Statement is an in-toto attestation statement payload.
type Statement struct {
	Type          string    `json:"_type"`
	Subject       []Subject `json:"subject"`
	PredicateType string    `json:"predicateType"`
	Predicate     any       `json:"predicate,omitempty"`
}

// Subject names an attested artifact and its digests.
type Subject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

// ParseBundle reads an attestation file that is either a bare Sigstore bundle
// or an npm attestations wrapper with one or more bundles.
func ParseBundle(path string) (Bundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Bundle{}, apperr.Wrap(apperr.IO, "provenance.parse", path, err)
	}
	var bare Bundle
	if err := json.Unmarshal(data, &bare); err == nil && bare.DSSEEnvelope.Payload != "" {
		return bare, nil
	}
	var wrapped npmAttestationsFile
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return Bundle{}, apperr.Wrap(apperr.Integrity, "provenance.parse", path, err)
	}
	for _, att := range wrapped.Attestations {
		if att.Bundle.DSSEEnvelope.Payload != "" {
			return att.Bundle, nil
		}
	}
	return Bundle{}, apperr.New(apperr.Integrity, "provenance.parse", path, "no dsse envelope in attestations")
}
