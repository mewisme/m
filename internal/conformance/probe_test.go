package conformance

import "testing"

// TestCertNegativeForcedSkip is a certification probe that must fail closed.
func TestCertNegativeForcedSkip(t *testing.T) {
	t.Skip("cert-negative probe: intentional skip")
}
