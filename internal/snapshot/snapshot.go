// Package snapshot holds install history snapshot descriptors for time travel (0028).
package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

// SchemaVersion versions serialized Snapshot documents.
const SchemaVersion = 1

// Snapshot describes an immutable historical install state.
type Snapshot struct {
	SchemaVersion int       `json:"schemaVersion"`
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	GraphDigest   string    `json:"graphDigest"`
	PolicyDigest  string    `json:"policyDigest,omitempty"`
}

// Normalize fills schema version and checks required fields.
func (s *Snapshot) Normalize() error {
	if s == nil {
		return apperr.New(apperr.Internal, "snapshot.normalize", "snapshot", "nil snapshot")
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	if s.SchemaVersion != SchemaVersion {
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
