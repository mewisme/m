package mlock

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

type semanticPayload struct {
	Settings  Settings          `json:"settings"`
	Importers []ImporterSection `json:"importers"`
	Packages  []graph.Package   `json:"packages"`
	Edges     []graph.Edge      `json:"edges"`
}

// semanticChecksum returns lowercase hex SHA-256 of the semantic payload.
func semanticChecksum(d *Document) (string, error) {
	payload := semanticPayload{
		Settings:  d.Settings,
		Importers: d.Importers,
		Packages:  d.Packages,
		Edges:     d.Edges,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return "", apperr.Wrap(apperr.Lockfile, "mlock.checksum", "m.lock", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

// Encode normalizes, computes checksum, and encodes deterministic JSON.
func Encode(d *Document) ([]byte, error) {
	if d == nil {
		return nil, apperr.New(apperr.Lockfile, "mlock.encode", "m.lock", "nil document")
	}
	d.LockfileVersion = LockfileVersion
	if err := d.Normalize(); err != nil {
		return nil, err
	}
	checksum, err := semanticChecksum(d)
	if err != nil {
		return nil, err
	}
	d.Checksum = checksum

	out := &Document{
		LockfileVersion: d.LockfileVersion,
		Checksum:        d.Checksum,
		Settings:        d.Settings,
		Importers:       d.Importers,
		Packages:        d.Packages,
		Edges:           d.Edges,
	}
	if len(d.Extensions) > 0 {
		out.Extensions = d.Extensions
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "mlock.encode", "m.lock", err)
	}
	return buf.Bytes(), nil
}

func verifyChecksum(d *Document) error {
	want := d.Checksum
	if want == "" {
		return apperr.New(apperr.Lockfile, "mlock.decode", "m.lock", "missing checksum")
	}
	got, err := semanticChecksum(d)
	if err != nil {
		return err
	}
	if got != want {
		return apperr.New(apperr.Lockfile, "mlock.decode", "m.lock",
			fmt.Sprintf("checksum mismatch: want %s got %s", want, got))
	}
	return nil
}
