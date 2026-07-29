package provenance

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fetch"
)

// TarballDigest is the expected tarball integrity for attestation subject matching.
type TarballDigest struct {
	Algo string // sha512
	Hex  string // lowercase hex
}

// VerifyResult summarizes a successful provenance verification.
type VerifyResult struct {
	PackageName      string `json:"packageName,omitempty"`
	PackageVersion   string `json:"packageVersion,omitempty"`
	SubjectPURL      string `json:"subjectPurl,omitempty"`
	DigestAlgo       string `json:"digestAlgo,omitempty"`
	DigestHex        string `json:"digestHex,omitempty"`
	Ecosystem        string `json:"ecosystem,omitempty"`
	TrustSource      string `json:"trustSource,omitempty"`
	VerificationMode string `json:"verificationMode,omitempty"`
}

// VerifyOptions configures attestation verification.
type VerifyOptions struct {
	TrustPolicy      TrustPolicy
	TrustedPublicKey ed25519.PublicKey
	Binding          *PackageBinding
}

// Verify checks a Sigstore bundle attestation and optionally matches tarball digest.
func Verify(ctx context.Context, attestationPath string, want TarballDigest, opts VerifyOptions) (VerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return VerifyResult{}, err
	}
	pub, trustSource, err := resolveTrust(opts)
	if err != nil {
		return VerifyResult{}, apperr.Wrap(apperr.Config, "provenance.verify", attestationPath, err)
	}
	bundle, err := ParseBundle(attestationPath)
	if err != nil {
		return VerifyResult{}, err
	}
	if err := verifyBundle(bundle, pub); err != nil {
		return VerifyResult{}, apperr.New(apperr.Integrity, "provenance.verify", attestationPath, err.Error())
	}
	stmt, err := decodeStatement(bundle.DSSEEnvelope.Payload)
	if err != nil {
		return VerifyResult{}, apperr.Wrap(apperr.Integrity, "provenance.verify", attestationPath, err)
	}
	subj, mode, err := selectSubject(stmt, opts.Binding, want)
	if err != nil {
		return VerifyResult{}, apperr.Wrap(apperr.Integrity, "provenance.verify.subject", attestationPath, err)
	}
	res, err := resultFromSubject(subj)
	if err != nil {
		return VerifyResult{}, apperr.Wrap(apperr.Integrity, "provenance.verify", attestationPath, err)
	}
	if opts.Binding != nil {
		res.Ecosystem = opts.Binding.Ecosystem
	}
	res.TrustSource = trustSource
	res.VerificationMode = mode
	if want.Algo != "" || want.Hex != "" {
		if err := matchDigest(res, want); err != nil {
			return VerifyResult{}, apperr.Wrap(apperr.Integrity, "provenance.verify.digest", res.SubjectPURL, err)
		}
	}
	return res, nil
}

func verifyBundle(bundle Bundle, pub ed25519.PublicKey) error {
	if bundle.MediaType != "" && bundle.MediaType != mediaTypeBundleV03 {
		return fmt.Errorf("unsupported media type %q", bundle.MediaType)
	}
	if bundle.VerificationMaterial.PublicKey != nil {
		raw, err := base64.StdEncoding.DecodeString(bundle.VerificationMaterial.PublicKey.RawBytes)
		if err != nil {
			return fmt.Errorf("decode bundle public key: %w", err)
		}
		if len(raw) != ed25519.PublicKeySize || !ed25519.PublicKey(raw).Equal(pub) {
			return fmt.Errorf("bundle public key does not match trusted public key")
		}
	}
	return verifyDSSE(bundle.DSSEEnvelope, pub)
}

func decodeStatement(payloadB64 string) (Statement, error) {
	raw, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		return Statement{}, err
	}
	var stmt Statement
	if err := json.Unmarshal(raw, &stmt); err != nil {
		return Statement{}, err
	}
	if stmt.Type != statementTypeV1 {
		return Statement{}, fmt.Errorf("unsupported statement type %q", stmt.Type)
	}
	if len(stmt.Subject) == 0 {
		return Statement{}, fmt.Errorf("statement has no subjects")
	}
	return stmt, nil
}

func selectSubject(stmt Statement, bind *PackageBinding, want TarballDigest) (Subject, string, error) {
	if bind != nil {
		subj, err := findMatchingSubject(stmt, *bind)
		return subj, "binding", err
	}
	if want.Algo != "" || want.Hex != "" {
		var matches []Subject
		for _, subj := range stmt.Subject {
			if subjectMatchesDigest(subj, want) {
				matches = append(matches, subj)
			}
		}
		switch len(matches) {
		case 0:
			return Subject{}, "", fmt.Errorf("no attestation subject matches expected digest")
		case 1:
			return matches[0], "digest-only", nil
		default:
			return Subject{}, "", fmt.Errorf("ambiguous attestation subjects for expected digest")
		}
	}
	if len(stmt.Subject) == 1 {
		return stmt.Subject[0], "signature-only", nil
	}
	return Subject{}, "", fmt.Errorf("package binding required when attestation has %d subjects", len(stmt.Subject))
}

func findMatchingSubject(stmt Statement, bind PackageBinding) (Subject, error) {
	var matches []Subject
	for _, subj := range stmt.Subject {
		if subjectMatchesBinding(subj, bind) {
			matches = append(matches, subj)
		}
	}
	switch len(matches) {
	case 0:
		target := bind.PURL
		if target == "" {
			target = bind.Name + "@" + bind.Version
		}
		return Subject{}, fmt.Errorf("no attestation subject matches package %s", target)
	case 1:
		return matches[0], nil
	default:
		return Subject{}, fmt.Errorf("ambiguous attestation subjects for package %s", bind.PURL)
	}
}

func subjectMatchesBinding(subj Subject, bind PackageBinding) bool {
	name, version := purlNameVersion(subj.Name)
	if bind.PURL != "" && subj.Name != bind.PURL {
		if name != bind.Name || version != bind.Version {
			return false
		}
	}
	if bind.Name != "" && name != bind.Name {
		return false
	}
	if bind.Version != "" && version != bind.Version {
		return false
	}
	if bind.Digest.Algo != "" || bind.Digest.Hex != "" {
		return subjectMatchesDigest(subj, bind.Digest)
	}
	return true
}

func subjectMatchesDigest(subj Subject, want TarballDigest) bool {
	algo, hex, err := subjectDigest(subj)
	if err != nil {
		return false
	}
	wantAlgo := strings.ToLower(strings.TrimSpace(want.Algo))
	wantHex := strings.ToLower(strings.TrimSpace(want.Hex))
	if wantAlgo == "" && wantHex != "" {
		wantAlgo = algo
	}
	if wantHex == "" {
		return false
	}
	return wantAlgo == algo && wantHex == hex
}

func resultFromSubject(subj Subject) (VerifyResult, error) {
	algo, hex, err := subjectDigest(subj)
	if err != nil {
		return VerifyResult{}, err
	}
	name, version := purlNameVersion(subj.Name)
	return VerifyResult{
		PackageName:    name,
		PackageVersion: version,
		SubjectPURL:    subj.Name,
		DigestAlgo:     algo,
		DigestHex:      hex,
		Ecosystem:      npmEcosystem,
	}, nil
}

func subjectDigest(subj Subject) (algo, hex string, err error) {
	if len(subj.Digest) == 0 {
		return "", "", fmt.Errorf("subject missing digest")
	}
	if h, ok := subj.Digest["sha512"]; ok {
		return "sha512", strings.ToLower(h), nil
	}
	for algo, hex := range subj.Digest {
		return algo, strings.ToLower(hex), nil
	}
	return "", "", fmt.Errorf("subject missing digest")
}

func matchDigest(res VerifyResult, want TarballDigest) error {
	wantAlgo := strings.ToLower(strings.TrimSpace(want.Algo))
	wantHex := strings.ToLower(strings.TrimSpace(want.Hex))
	if wantAlgo == "" && wantHex != "" {
		wantAlgo = res.DigestAlgo
	}
	if wantHex == "" {
		return fmt.Errorf("missing expected digest")
	}
	if wantAlgo != res.DigestAlgo {
		return fmt.Errorf("digest algo mismatch: attestation %s expected %s", res.DigestAlgo, wantAlgo)
	}
	if wantHex != res.DigestHex {
		return fmt.Errorf("digest mismatch: attestation %s expected %s", res.DigestHex, wantHex)
	}
	return nil
}

func purlNameVersion(purl string) (name, version string) {
	const prefix = "pkg:npm/"
	if !strings.HasPrefix(purl, prefix) {
		return "", ""
	}
	rest := strings.TrimPrefix(purl, prefix)
	at := strings.LastIndex(rest, "@")
	if at < 0 {
		return rest, ""
	}
	return rest[:at], rest[at+1:]
}

// DigestFromIntegrity parses lockfile integrity into a TarballDigest.
func DigestFromIntegrity(integrity string) (TarballDigest, error) {
	parsed, err := fetch.ParseIntegrity(integrity)
	if err != nil {
		return TarballDigest{}, err
	}
	return TarballDigest{Algo: parsed.Algo, Hex: parsed.Hex}, nil
}
