package policy_test

import (
	"bytes"
	"testing"

	"github.com/mewisme/m/internal/policy"
)

func TestPolicyRoundTrip(t *testing.T) {
	p := &policy.Policy{
		SchemaVersion:          policy.SchemaVersion,
		ScriptTrust:            policy.ScriptTrustDeny,
		Offline:                true,
		Linker:                 "isolated",
		StrictPeerDependencies: true,
	}
	first, err := policy.EncodeJSON(p)
	if err != nil {
		t.Fatal(err)
	}
	again, err := policy.DecodeJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := policy.EncodeJSON(again)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("unstable\n%s\n%s", first, second)
	}
}
