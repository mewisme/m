package transaction

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/mewisme/m/internal/apperr"
)

const (
	SchemaVersion = 1
	JournalName   = "journal.v1.json"
)

// Document states.
const (
	StateStaging    = "staging"
	StateValidated  = "validated"
	StateCommitting = "committing"
	StateCommitted  = "committed"
	StateAborted    = "aborted"
)

// Op kinds.
const (
	OpBackup = "backup"
	OpRename = "rename"
	OpWrite  = "write"
	OpRemove = "remove"
	OpMkdir  = "mkdir"
)

// Document is journal.v1 — deterministic, versioned.
type Document struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	ProjectRoot   string `json:"projectRoot"`
	State         string `json:"state"`
	Ops           []Op   `json:"ops"`
}

// Op is one journaled filesystem mutation with inverse metadata.
type Op struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Backup string `json:"backup,omitempty"`
}

// Encode normalizes and encodes doc to JSON with trailing newline.
func Encode(doc *Document) ([]byte, error) {
	if err := Normalize(doc); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, apperr.Wrap(apperr.Transaction, "transaction.encode", JournalName, err)
	}
	return buf.Bytes(), nil
}

// Decode unmarshals and normalizes a journal document.
func Decode(data []byte) (*Document, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, apperr.Wrap(apperr.Transaction, "transaction.decode", JournalName, err)
	}
	if err := Normalize(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// Normalize fills defaults and enforces stable op ordering.
func Normalize(doc *Document) error {
	if doc == nil {
		return apperr.New(apperr.Transaction, "transaction.normalize", JournalName, "nil document")
	}
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = SchemaVersion
	}
	if doc.SchemaVersion != SchemaVersion {
		return apperr.New(apperr.Transaction, "transaction.normalize", JournalName,
			fmt.Sprintf("unsupported schemaVersion %d", doc.SchemaVersion))
	}
	if doc.ID == "" {
		return apperr.New(apperr.Transaction, "transaction.normalize", JournalName, "empty id")
	}
	if doc.ProjectRoot == "" {
		return apperr.New(apperr.Transaction, "transaction.normalize", JournalName, "empty projectRoot")
	}
	if doc.State == "" {
		doc.State = StateStaging
	}
	return nil
}
