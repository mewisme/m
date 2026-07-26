// Package releasetrain validates the MVP dependency graph and release-train rules.
package releasetrain

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// SchemaVersion is the milestones.json schema version.
const SchemaVersion = 1

// Milestone is one MVP node in the release train.
type Milestone struct {
	ID           string   `json:"id"`
	Phase        string   `json:"phase"`
	Predecessors []string `json:"predecessors"`
}

// Graph is the full milestone dependency document.
type Graph struct {
	SchemaVersion int         `json:"schemaVersion"`
	Milestones    []Milestone `json:"milestones"`
}

// LoadFile reads and validates a milestones.json document.
func LoadFile(path string) (*Graph, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Load(b)
}

// Load parses milestones JSON bytes.
func Load(b []byte) (*Graph, error) {
	var g Graph
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, fmt.Errorf("releasetrain: parse: %w", err)
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}

// Validate checks schema, IDs, predecessor references, and acyclicity.
func (g *Graph) Validate() error {
	if g == nil {
		return fmt.Errorf("releasetrain: nil graph")
	}
	if g.SchemaVersion != SchemaVersion {
		return fmt.Errorf("releasetrain: unsupported schemaVersion %d", g.SchemaVersion)
	}
	if len(g.Milestones) == 0 {
		return fmt.Errorf("releasetrain: no milestones")
	}
	byID := make(map[string]*Milestone, len(g.Milestones))
	for i := range g.Milestones {
		m := &g.Milestones[i]
		if m.ID == "" {
			return fmt.Errorf("releasetrain: milestones[%d]: empty id", i)
		}
		if _, dup := byID[m.ID]; dup {
			return fmt.Errorf("releasetrain: duplicate id %q", m.ID)
		}
		if m.Predecessors == nil {
			m.Predecessors = []string{}
		}
		byID[m.ID] = m
	}
	for _, m := range g.Milestones {
		for _, p := range m.Predecessors {
			if _, ok := byID[p]; !ok {
				return fmt.Errorf("releasetrain: %s predecessor %q missing", m.ID, p)
			}
		}
	}
	if err := g.Acyclic(); err != nil {
		return err
	}
	return nil
}

// IDs returns sorted milestone IDs.
func (g *Graph) IDs() []string {
	ids := make([]string, len(g.Milestones))
	for i, m := range g.Milestones {
		ids[i] = m.ID
	}
	sort.Strings(ids)
	return ids
}

// ByID returns a milestone or false.
func (g *Graph) ByID(id string) (Milestone, bool) {
	for _, m := range g.Milestones {
		if m.ID == id {
			return m, true
		}
	}
	return Milestone{}, false
}

// Acyclic reports a cycle if present.
func (g *Graph) Acyclic() error {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(g.Milestones))
	preds := make(map[string][]string, len(g.Milestones))
	for _, m := range g.Milestones {
		preds[m.ID] = append([]string(nil), m.Predecessors...)
		color[m.ID] = white
	}
	var stack []string
	var visit func(string) error
	visit = func(n string) error {
		color[n] = gray
		stack = append(stack, n)
		for _, p := range preds[n] {
			switch color[p] {
			case gray:
				return fmt.Errorf("releasetrain: cycle involving %s -> %s (stack %v)", n, p, stack)
			case white:
				if err := visit(p); err != nil {
					return err
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = black
		return nil
	}
	ids := g.IDs()
	for _, id := range ids {
		if color[id] == white {
			if err := visit(id); err != nil {
				return err
			}
		}
	}
	return nil
}

// TransitivePredecessors returns all ancestors of id (not including id).
func (g *Graph) TransitivePredecessors(id string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	var walk func(string) error
	walk = func(cur string) error {
		m, ok := g.ByID(cur)
		if !ok {
			return fmt.Errorf("releasetrain: unknown id %q", cur)
		}
		for _, p := range m.Predecessors {
			if _, seen := out[p]; seen {
				continue
			}
			out[p] = struct{}{}
			if err := walk(p); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(id); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidateStabilizationOrder enforces 0031 before runners and 0046 before runtime.
func (g *Graph) ValidateStabilizationOrder() error {
	anc40, err := g.TransitivePredecessors("0040")
	if err != nil {
		return err
	}
	if _, ok := anc40["0031"]; !ok {
		return fmt.Errorf("releasetrain: 0040 must transitively require 0031")
	}
	anc50, err := g.TransitivePredecessors("0050")
	if err != nil {
		return err
	}
	if _, ok := anc50["0046"]; !ok {
		return fmt.Errorf("releasetrain: 0050 must transitively require 0046")
	}
	anc31, err := g.TransitivePredecessors("0031")
	if err != nil {
		return err
	}
	for id := range anc31 {
		if id >= "0040" && id < "0080" {
			// runners 0040-0046 and runtime 0050-0057 must not precede core gate
			if (id >= "0040" && id <= "0046") || (id >= "0050" && id <= "0057") {
				return fmt.Errorf("releasetrain: 0031 must not depend on runner/runtime %s", id)
			}
		}
	}
	return nil
}

// ValidateNonBlocking0090 ensures 0090 is not required by stabilization gates.
func (g *Graph) ValidateNonBlocking0090() error {
	for _, gate := range []string{"0031", "0046", "0057"} {
		anc, err := g.TransitivePredecessors(gate)
		if err != nil {
			return err
		}
		if _, ok := anc["0090"]; ok {
			return fmt.Errorf("releasetrain: gate %s must not require 0090", gate)
		}
		m, ok := g.ByID(gate)
		if !ok {
			return fmt.Errorf("releasetrain: missing gate %s", gate)
		}
		for _, p := range m.Predecessors {
			if p == "0090" {
				return fmt.Errorf("releasetrain: gate %s must not list 0090 as predecessor", gate)
			}
		}
	}
	m90, ok := g.ByID("0090")
	if !ok {
		return fmt.Errorf("releasetrain: missing 0090")
	}
	has87 := false
	for _, p := range m90.Predecessors {
		if p == "0087" {
			has87 = true
		}
	}
	if !has87 {
		return fmt.Errorf("releasetrain: 0090 must require 0087")
	}
	return nil
}
