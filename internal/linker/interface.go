// Package linker plans and applies hoisted and isolated node_modules layouts.
package linker

import (
	"context"
	"strings"

	"github.com/mewisme/mew/internal/graph"
)

// OpKind classifies one filesystem operation in a link plan.
type OpKind string

const (
	OpMkdir    OpKind = "mkdir"
	OpCopy     OpKind = "copy"
	OpHardlink OpKind = "hardlink"
	OpReflink  OpKind = "reflink"
	OpSymlink  OpKind = "symlink"
	OpJunction OpKind = "junction"
)

// Op is one mkdir or recursive directory copy step.
type Op struct {
	Kind OpKind `json:"kind"`
	Src  string `json:"src,omitempty"`
	Dest string `json:"dest,omitempty"`
}

// PlacementID uniquely identifies one physical install instance in a hoisted layout.
// Components: parentPlacement | importer | depName | packageKey | hoistLevel | peerContext.
type PlacementID struct {
	Parent      string `json:"parent,omitempty"`
	Importer    string `json:"importer"`
	DepName     string `json:"depName"`
	PackageKey  string `json:"packageKey"`
	HoistLevel  int    `json:"hoistLevel"`
	PeerContext string `json:"peerContext,omitempty"`
}

// String returns a stable serialization for sorting and cycle detection.
func (id PlacementID) String() string {
	return id.Parent + "|" + id.Importer + "|" + id.DepName + "|" + id.PackageKey + "|" + itoa(id.HoistLevel) + "|" + id.PeerContext
}

// Compare returns -1, 0, or 1 for deterministic placement ordering.
func (id PlacementID) Compare(other PlacementID) int {
	return strings.Compare(id.String(), other.String())
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// Placement records where one resolved package is installed under node_modules.
type Placement struct {
	ID      PlacementID `json:"id"`
	Key     string      `json:"key"`
	DestDir string      `json:"destDir"`
}

// BinSource describes one command to expose under a node_modules/.bin directory.
type BinSource struct {
	Cmd         string `json:"cmd"`
	Target      string `json:"target"` // script path relative to package root
	PackageDir  string `json:"packageDir"`
	NodeModules string `json:"nodeModules,omitempty"` // defaults to plan.NodeModules
}

// LinkSummary tallies filesystem strategies used in a plan.
type LinkSummary struct {
	Mkdir    int `json:"mkdir,omitempty"`
	Copy     int `json:"copy,omitempty"`
	Hardlink int `json:"hardlink,omitempty"`
	Reflink  int `json:"reflink,omitempty"`
	Symlink  int `json:"symlink,omitempty"`
	Junction int `json:"junction,omitempty"`
}

// TallyFromOps counts op kinds into s (mutates s).
func (s *LinkSummary) TallyFromOps(ops []Op) {
	if s == nil {
		return
	}
	for _, op := range ops {
		switch op.Kind {
		case OpMkdir:
			s.Mkdir++
		case OpCopy:
			s.Copy++
		case OpHardlink:
			s.Hardlink++
		case OpReflink:
			s.Reflink++
		case OpSymlink:
			s.Symlink++
		case OpJunction:
			s.Junction++
		}
	}
}

// Plan is a filesystem link plan (mkdir/copy ops and bin shims).
// Distinct from plan.Plan, which is the install mutation plan.
type Plan struct {
	LayoutMode  string            `json:"layoutMode,omitempty"` // hoisted | isolated
	NodeModules string            `json:"nodeModules"`
	ExtractDirs map[string]string `json:"extractDirs,omitempty"`
	Placements  []Placement       `json:"placements,omitempty"`
	Ops         []Op              `json:"ops,omitempty"`
	Bins        []BinSource       `json:"bins,omitempty"`
	LinkSummary LinkSummary       `json:"linkSummary,omitempty"`
}

// Linker plans link operations and applies them under a transaction stage.
type Linker interface {
	Plan(ctx context.Context, g *graph.Graph) (*Plan, error)
	Apply(ctx context.Context, plan *Plan) error
}
