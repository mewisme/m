package provenance

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

func dssePAE(payloadType, payload string) []byte {
	return []byte(fmt.Sprintf("DSSEv1 %d %s %d %s", len(payloadType), payloadType, len(payload), payload))
}

func verifyDSSE(env DSSEEnvelope, pub ed25519.PublicKey) error {
	if env.PayloadType != payloadTypeInToto {
		return fmt.Errorf("unsupported payload type %q", env.PayloadType)
	}
	if len(env.Signatures) != 1 {
		return fmt.Errorf("expected 1 dsse signature, got %d", len(env.Signatures))
	}
	sig, err := base64.StdEncoding.DecodeString(env.Signatures[0].Sig)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	pae := dssePAE(env.PayloadType, env.Payload)
	if !ed25519.Verify(pub, pae, sig) {
		return fmt.Errorf("dsse signature mismatch")
	}
	return nil
}
