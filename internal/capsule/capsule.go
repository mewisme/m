// Package capsule holds portable dependency capsule descriptors (types only).
package capsule

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mewisme/m/internal/apperr"
)

// SchemaVersion versions serialized Capsule documents.
const SchemaVersion = 1

// Capsule describes a portable set of packages with integrity.
type Capsule struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	Packages      []string `json:"packages"` // package keys
	Integrity     string   `json:"integrity,omitempty"`
}

// Normalize sorts package keys and validates required fields.
func (c *Capsule) Normalize() error {
	if c == nil {
		return apperr.New(apperr.Internal, "capsule.normalize", "capsule", "nil capsule")
	}
	if c.SchemaVersion == 0 {
		c.SchemaVersion = SchemaVersion
	}
	if c.SchemaVersion != SchemaVersion {
		return apperr.New(apperr.Internal, "capsule.normalize", "capsule",
			fmt.Sprintf("unsupported schemaVersion %d", c.SchemaVersion))
	}
	if c.ID == "" {
		return apperr.New(apperr.Internal, "capsule.normalize", "capsule", "empty id")
	}
	if c.Packages == nil {
		c.Packages = []string{}
	}
	sort.Strings(c.Packages)
	return nil
}

// EncodeJSON normalizes and encodes with indent.
func EncodeJSON(c *Capsule) ([]byte, error) {
	if err := c.Normalize(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "capsule.encode", "capsule", err)
	}
	return buf.Bytes(), nil
}

// DecodeJSON unmarshals and normalizes a capsule.
func DecodeJSON(data []byte) (*Capsule, error) {
	var c Capsule
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "capsule.decode", "capsule", err)
	}
	if err := c.Normalize(); err != nil {
		return nil, err
	}
	return &c, nil
}
