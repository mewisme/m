package transaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/mewisme/m/internal/apperr"
)

const (
	SchemaVersion = 4
	JournalName   = "journal.v4.json"
	JournalNameV3 = "journal.v3.json"
	JournalNameV2 = "journal.v2.json"
	JournalNameV1 = "journal.v1.json"
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

// Op progress states.
const (
	ProgressPending     = "pending"
	ProgressApplying    = "applying"
	ProgressApplied     = "applied"
	ProgressRollingBack = "rolling_back"
	ProgressRolledBack  = "rolled_back"
)

// Op phase sub-states (journal v3 plan ops).
const (
	PhasePending          = "pending"
	PhasePriorIdentified  = "prior_identified"
	PhasePriorBackedUp    = "prior_backed_up"
	PhasePriorMovedAside  = "prior_moved_aside"
	PhasePublishStarted   = "publish_started"
	PhasePublished        = "published"
	PhaseApplied          = "applied"
	PhaseRollbackStarted  = "rollback_started"
	PhasePriorRestored    = "prior_restored"
	PhaseRollbackComplete = "rollback_completed"
)

// Destination kinds for backup metadata.
const (
	DestKindNone     = "none"
	DestKindFile     = "file"
	DestKindDir      = "dir"
	DestKindSymlink  = "symlink"
	DestKindJunction = "junction"
)

// Document is the versioned transaction journal.
type Document struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	ProjectRoot   string `json:"projectRoot"`
	State         string `json:"state"`
	Checksum      string `json:"checksum,omitempty"`
	Plan          []Op   `json:"plan,omitempty"`
	Ops           []Op   `json:"ops"`
}

// Op is one journaled filesystem mutation with inverse metadata.
type Op struct {
	Kind          string `json:"kind"`
	Path          string `json:"path"`
	Backup        string `json:"backup,omitempty"`
	Progress      string `json:"progress,omitempty"`
	Phase         string `json:"phase,omitempty"`
	StagingPath   string `json:"stagingPath,omitempty"`
	DestKind      string `json:"destKind,omitempty"`
	HadPrior      bool   `json:"hadPrior,omitempty"`
	PriorKind     string `json:"priorKind,omitempty"`
	SymlinkTarget string `json:"symlinkTarget,omitempty"`
}

// Encode normalizes and encodes doc to JSON with trailing newline.
func Encode(doc *Document) ([]byte, error) {
	if err := Normalize(doc); err != nil {
		return nil, err
	}
	if doc.SchemaVersion >= 4 {
		checksum, err := semanticChecksum(doc)
		if err != nil {
			return nil, err
		}
		doc.Checksum = checksum
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

func semanticChecksum(doc *Document) (string, error) {
	cp := *doc
	cp.Checksum = ""
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&cp); err != nil {
		return "", apperr.Wrap(apperr.Transaction, "transaction.checksum", JournalName, err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func verifyChecksum(doc *Document) error {
	if doc.SchemaVersion < 4 || doc.Checksum == "" {
		return nil
	}
	got, err := semanticChecksum(doc)
	if err != nil {
		return err
	}
	if got != doc.Checksum {
		return apperr.New(apperr.Integrity, "transaction.decode", JournalName, "checksum mismatch")
	}
	return nil
}

// Decode unmarshals and normalizes a journal document (v1–v4).
func Decode(data []byte) (*Document, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, apperr.Wrap(apperr.Transaction, "transaction.decode", JournalName, err)
	}
	if err := Normalize(&doc); err != nil {
		return nil, err
	}
	if err := verifyChecksum(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// Normalize fills defaults and enforces invariants.
func Normalize(doc *Document) error {
	if doc == nil {
		return apperr.New(apperr.Transaction, "transaction.normalize", JournalName, "nil document")
	}
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = SchemaVersion
	}
	if doc.SchemaVersion != 1 && doc.SchemaVersion != 2 && doc.SchemaVersion != 3 && doc.SchemaVersion != SchemaVersion {
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
	for i := range doc.Plan {
		if doc.Plan[i].Progress == "" {
			doc.Plan[i].Progress = ProgressPending
		}
		if doc.Plan[i].Phase == "" {
			doc.Plan[i].Phase = PhasePending
		}
	}
	return nil
}
