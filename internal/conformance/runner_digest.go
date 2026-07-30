package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// RunnerManifestDigest returns canonical SHA-256 hex for a validated manifest.
func RunnerManifestDigest(m RunnerManifest) (string, error) {
	cp := m
	sortRunnerManifestForDigest(&cp)
	return canonicalDigest(cp)
}

// RunnerWaiverManifestDigest returns canonical SHA-256 hex for validated waivers.
func RunnerWaiverManifestDigest(w WaiverManifest) (string, error) {
	cp := w
	sortWaiverManifestForDigest(&cp)
	return canonicalDigest(cp)
}

func canonicalDigest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
