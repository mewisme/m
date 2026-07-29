package capsule

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/contentid"
)

// SchemaVersion versions serialized capsule documents.
const SchemaVersion = 1

// Capsule describes a portable set of packages with integrity.
type Capsule struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	Packages      []string `json:"packages"` // package keys
	Integrity     string   `json:"integrity,omitempty"`
}

// BlobRef identifies one content-addressed tarball blob in a capsule archive.
type BlobRef struct {
	Algo string `json:"algo"`
	Hex  string `json:"hex"`
}

// Manifest is the portable export descriptor embedded in capsule archives.
type Manifest struct {
	SchemaVersion   int               `json:"schemaVersion"`
	ID              string            `json:"id"`
	CreatedAt       time.Time         `json:"createdAt"`
	GraphDigest     string            `json:"graphDigest"`
	PolicyDigest    string            `json:"policyDigest,omitempty"`
	Platform        string            `json:"platform"`
	NodeVersion     string            `json:"nodeVersion,omitempty"`
	Lock            []byte            `json:"lock"`
	Manifest        []byte            `json:"manifest"`
	MemberManifests map[string][]byte `json:"memberManifests,omitempty"`
	Blobs           []BlobRef         `json:"blobs"`
	Integrity       string            `json:"integrity"`
}

// Normalize sorts blob refs and validates required fields.
func (m *Manifest) Normalize() error {
	if m == nil {
		return apperr.New(apperr.Internal, "capsule.normalize", "manifest", "nil manifest")
	}
	if m.SchemaVersion == 0 {
		m.SchemaVersion = SchemaVersion
	}
	if m.SchemaVersion != SchemaVersion {
		return apperr.New(apperr.Internal, "capsule.normalize", "manifest",
			fmt.Sprintf("unsupported schemaVersion %d", m.SchemaVersion))
	}
	if m.ID == "" {
		return apperr.New(apperr.Internal, "capsule.normalize", "manifest", "empty id")
	}
	if m.Platform == "" {
		return apperr.New(apperr.Internal, "capsule.normalize", "manifest", "empty platform")
	}
	if len(m.Lock) == 0 {
		return apperr.New(apperr.Internal, "capsule.normalize", "manifest", "empty lock")
	}
	if len(m.Manifest) == 0 {
		return apperr.New(apperr.Internal, "capsule.normalize", "manifest", "empty package manifest")
	}
	if m.Blobs == nil {
		m.Blobs = []BlobRef{}
	}
	sort.Slice(m.Blobs, func(i, j int) bool {
		if m.Blobs[i].Algo != m.Blobs[j].Algo {
			return m.Blobs[i].Algo < m.Blobs[j].Algo
		}
		return m.Blobs[i].Hex < m.Blobs[j].Hex
	})
	for _, ref := range m.Blobs {
		if err := contentid.ValidateKey(ref.Algo, ref.Hex); err != nil {
			return apperr.Wrap(apperr.Integrity, "capsule.normalize", ref.Algo+"/"+ref.Hex, err)
		}
	}
	if m.MemberManifests == nil {
		m.MemberManifests = map[string][]byte{}
	}
	return nil
}

// ComputeID returns a content-addressed capsule id from lock bytes and platform.
func ComputeID(lock []byte, platform string) string {
	sum := sha256.Sum256(lock)
	payload := platform + "\n" + hex.EncodeToString(sum[:])
	idSum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(idSum[:])
}

// ComputeIntegrity hashes the manifest payload excluding the integrity field.
func ComputeIntegrity(m *Manifest) (string, error) {
	if m == nil {
		return "", apperr.New(apperr.Internal, "capsule.integrity", "manifest", "nil manifest")
	}
	clone := *m
	clone.Integrity = ""
	if err := clone.Normalize(); err != nil {
		return "", err
	}
	data, err := encodeManifestJSON(&clone)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// SealIntegrity normalizes the manifest and sets Integrity.
func SealIntegrity(m *Manifest) error {
	if err := m.Normalize(); err != nil {
		return err
	}
	integrity, err := ComputeIntegrity(m)
	if err != nil {
		return err
	}
	m.Integrity = integrity
	return nil
}

// VerifyIntegrity recomputes and checks the manifest integrity field.
func VerifyIntegrity(m *Manifest) error {
	if m == nil {
		return apperr.New(apperr.Internal, "capsule.verify", "manifest", "nil manifest")
	}
	want := m.Integrity
	got, err := ComputeIntegrity(m)
	if err != nil {
		return err
	}
	if want == "" {
		return apperr.New(apperr.Integrity, "capsule.verify", "manifest", "missing integrity")
	}
	if got != want {
		return apperr.New(apperr.Integrity, "capsule.verify", "manifest",
			fmt.Sprintf("integrity mismatch: got %s want %s", got, want))
	}
	return nil
}

// BlobPath returns blobs/<algo>/<hex> for archive members.
func (r BlobRef) BlobPath() string {
	return "blobs/" + r.Algo + "/" + r.Hex
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

// EncodeManifestJSON normalizes and encodes a manifest with indent.
func EncodeManifestJSON(m *Manifest) ([]byte, error) {
	if err := m.Normalize(); err != nil {
		return nil, err
	}
	return encodeManifestJSON(m)
}

// DecodeManifestJSON unmarshals and normalizes a manifest.
func DecodeManifestJSON(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "capsule.decode", "manifest", err)
	}
	if err := m.Normalize(); err != nil {
		return nil, err
	}
	return &m, nil
}

func encodeManifestJSON(m *Manifest) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "capsule.encode", "manifest", err)
	}
	return buf.Bytes(), nil
}
