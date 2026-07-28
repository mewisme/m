// Package plan holds install mutation plan models (desired state, operations, commits).
// Distinct from linker.Plan, which describes filesystem link operations only.
package plan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
)

// SchemaVersion versions serialized Plan documents.
const SchemaVersion = 1

// DesiredState is one package that should exist after commit.
type DesiredState struct {
	PackageKey string `json:"packageKey"`
	Integrity  string `json:"integrity,omitempty"`
}

// Operation is one physical install-family action.
type Operation struct {
	Op      string `json:"op"`
	Subject string `json:"subject"`
	Detail  string `json:"detail,omitempty"`
}

// CommitAction is a final commit-phase step (swap roots, write lock, etc.).
type CommitAction struct {
	Op      string `json:"op"`
	Subject string `json:"subject"`
	Detail  string `json:"detail,omitempty"`
}

// Plan is a serializable install mutation plan for 0028 explain/preview.
type Plan struct {
	SchemaVersion int            `json:"schemaVersion"`
	Desired       []DesiredState `json:"desired"`
	Operations    []Operation    `json:"operations"`
	Commits       []CommitAction `json:"commits"`
}

// Normalize sorts collections for deterministic encoding.
func (p *Plan) Normalize() error {
	if p == nil {
		return apperr.New(apperr.Internal, "plan.normalize", "plan", "nil plan")
	}
	if p.SchemaVersion == 0 {
		p.SchemaVersion = SchemaVersion
	}
	if p.SchemaVersion != SchemaVersion {
		return apperr.New(apperr.Internal, "plan.normalize", "plan",
			fmt.Sprintf("unsupported schemaVersion %d", p.SchemaVersion))
	}
	if p.Desired == nil {
		p.Desired = []DesiredState{}
	}
	if p.Operations == nil {
		p.Operations = []Operation{}
	}
	if p.Commits == nil {
		p.Commits = []CommitAction{}
	}
	sort.SliceStable(p.Desired, func(i, j int) bool {
		return p.Desired[i].PackageKey < p.Desired[j].PackageKey
	})
	sort.SliceStable(p.Operations, func(i, j int) bool {
		a, b := p.Operations[i], p.Operations[j]
		if a.Op != b.Op {
			return a.Op < b.Op
		}
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		return a.Detail < b.Detail
	})
	sort.SliceStable(p.Commits, func(i, j int) bool {
		a, b := p.Commits[i], p.Commits[j]
		if a.Op != b.Op {
			return a.Op < b.Op
		}
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		return a.Detail < b.Detail
	})
	return nil
}

// EncodeJSON normalizes and encodes with indent.
func EncodeJSON(p *Plan) ([]byte, error) {
	if err := p.Normalize(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "plan.encode", "plan", err)
	}
	return buf.Bytes(), nil
}

// DecodeJSON unmarshals and normalizes a plan.
func DecodeJSON(data []byte) (*Plan, error) {
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "plan.decode", "plan", err)
	}
	if err := p.Normalize(); err != nil {
		return nil, err
	}
	return &p, nil
}
