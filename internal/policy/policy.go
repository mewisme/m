package policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

// SchemaVersion versions serialized Policy documents.
const SchemaVersion = 1

// ScriptTrust controls lifecycle script execution.
type ScriptTrust string

const (
	ScriptTrustAllow ScriptTrust = "allow"
	ScriptTrustDeny  ScriptTrust = "deny"
	ScriptTrustAsk   ScriptTrust = "ask"
)

// Policy is the trust/sandbox descriptor consumed by resolve and install.
type Policy struct {
	SchemaVersion     int           `json:"schemaVersion"`
	ScriptTrust       ScriptTrust   `json:"scriptTrust"`
	Offline           bool          `json:"offline,omitempty"`
	Linker            string        `json:"linker,omitempty"`            // auto|hoisted|isolated
	MinimumReleaseAge time.Duration `json:"minimumReleaseAge,omitempty"` // 0 = off
	RejectDeprecated  bool          `json:"rejectDeprecated,omitempty"`
}

// Normalize fills defaults and validates enums.
func (p *Policy) Normalize() error {
	if p == nil {
		return apperr.New(apperr.Config, "policy.normalize", "policy", "nil policy")
	}
	if p.SchemaVersion == 0 {
		p.SchemaVersion = SchemaVersion
	}
	if p.SchemaVersion != SchemaVersion {
		return apperr.New(apperr.Config, "policy.normalize", "policy",
			fmt.Sprintf("unsupported schemaVersion %d", p.SchemaVersion))
	}
	if p.ScriptTrust == "" {
		p.ScriptTrust = ScriptTrustAsk
	}
	switch p.ScriptTrust {
	case ScriptTrustAllow, ScriptTrustDeny, ScriptTrustAsk:
	default:
		return apperr.New(apperr.Config, "policy.normalize", "policy",
			fmt.Sprintf("unknown scriptTrust %q", p.ScriptTrust))
	}
	if p.Linker != "" {
		switch p.Linker {
		case "auto", "hoisted", "isolated":
		default:
			return apperr.New(apperr.Config, "policy.normalize", "policy",
				fmt.Sprintf("unknown linker %q", p.Linker))
		}
	}
	return nil
}

// EncodeJSON normalizes and encodes with indent.
func EncodeJSON(p *Policy) ([]byte, error) {
	if err := p.Normalize(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		return nil, apperr.Wrap(apperr.Config, "policy.encode", "policy", err)
	}
	return buf.Bytes(), nil
}

// DecodeJSON unmarshals and normalizes a policy.
func DecodeJSON(data []byte) (*Policy, error) {
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, apperr.Wrap(apperr.Config, "policy.decode", "policy", err)
	}
	if err := p.Normalize(); err != nil {
		return nil, err
	}
	return &p, nil
}
