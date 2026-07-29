// Package provenance verifies npm Sigstore provenance attestation bundles.
//
// Production callers must supply an explicit TrustPolicy; fixture keys are
// test-only. Package identity binding requires an exact name@version match
// across all statement subjects.
package provenance
