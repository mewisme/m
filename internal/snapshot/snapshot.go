// Package snapshot holds install history snapshot descriptors for time travel (0028).
package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// SchemaVersion is the current on-disk snapshot metadata version.
const SchemaVersion = 2

// SchemaVersionV1 is the root-only snapshot format.
const SchemaVersionV1 = 1

// Snapshot describes an immutable historical install state.
type Snapshot struct {
	SchemaVersion   int       `json:"schemaVersion"`
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	GraphDigest     string    `json:"graphDigest"`
	PolicyDigest    string    `json:"policyDigest,omitempty"`
	MemberManifests []string  `json:"memberManifests,omitempty"`
}

// Normalize fills schema version and checks required fields.
func (s *Snapshot) Normalize() error {
	if s == nil {
		return apperr.New(apperr.Internal, "snapshot.normalize", "snapshot", "nil snapshot")
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersionV1
	}
	if s.SchemaVersion != SchemaVersion && s.SchemaVersion != SchemaVersionV1 {
		return apperr.New(apperr.Internal, "snapshot.normalize", "snapshot",
			fmt.Sprintf("unsupported schemaVersion %d", s.SchemaVersion))
	}
	if s.ID == "" {
		return apperr.New(apperr.Internal, "snapshot.normalize", "snapshot", "empty id")
	}
	if s.GraphDigest == "" {
		return apperr.New(apperr.Internal, "snapshot.normalize", "snapshot", "empty graphDigest")
	}
	return nil
}

// EncodeJSON normalizes and encodes with indent.
func EncodeJSON(s *Snapshot) ([]byte, error) {
	if err := s.Normalize(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "snapshot.encode", "snapshot", err)
	}
	return buf.Bytes(), nil
}

// DecodeJSON unmarshals and normalizes a snapshot.
func DecodeJSON(data []byte) (*Snapshot, error) {
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "snapshot.decode", "snapshot", err)
	}
	if err := s.Normalize(); err != nil {
		return nil, err
	}
	return &s, nil
}
