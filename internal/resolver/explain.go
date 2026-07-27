package resolver

import (
	"context"
	"errors"

	"github.com/mewisme/m/internal/graph"
)

// ConflictNode is one node in a peer conflict explanation tree.
type ConflictNode struct {
	Constraint string         `json:"constraint,omitempty"`
	Importer   string         `json:"importer,omitempty"`
	Candidates []string       `json:"candidates,omitempty"`
	Children   []ConflictNode `json:"children,omitempty"`
}

// ConflictTree explains an unsatisfiable peer dependency.
type ConflictTree struct {
	Peer string       `json:"peer"`
	Root ConflictNode `json:"root"`
}

// PeerConflictFromError extracts a structured peer conflict from a resolve error.
func PeerConflictFromError(err error) (PeerConflict, bool) {
	var pc *peerConflictErr
	if errors.As(err, &pc) {
		return pc.PeerConflict, true
	}
	return PeerConflict{}, false
}

// BuildPeerConflictTree builds a human/machine conflict tree for a peer failure.
func BuildPeerConflictTree(conf PeerConflict, prior *graph.Graph) ConflictTree {
	tree := ConflictTree{
		Peer: conf.Peer,
		Root: ConflictNode{
			Constraint: conf.Peer + "@" + conf.Range,
			Importer:   conf.Package,
			Candidates: peerCandidates(prior, conf.Peer, conf.Range),
		},
	}
	return tree
}

func peerCandidates(g *graph.Graph, peerName, rng string) []string {
	if g == nil {
		return nil
	}
	var out []string
	for _, p := range g.Packages {
		if p.ID.Name != peerName {
			continue
		}
		out = append(out, p.ID.Version)
	}
	if len(out) == 0 {
		out = append(out, "(none resolved)")
	}
	return out
}

// ExplainPeer dry-resolves and returns a conflict tree when peerName is unsatisfied.
func (e *Engine) ExplainPeer(ctx context.Context, root, peerName string, opts ResolveOptions) (*ConflictTree, error) {
	res, err := e.Resolve(ctx, root, opts)
	if err == nil {
		for _, p := range res.Graph.Packages {
			if p.ID.Name == peerName {
				return nil, nil
			}
		}
		return nil, nil
	}
	conf, ok := PeerConflictFromError(err)
	if !ok || conf.Peer != peerName {
		return nil, err
	}
	prior := opts.Prior
	if prior == nil {
		prior = opts.Hints
	}
	tree := BuildPeerConflictTree(conf, prior)
	return &tree, nil
}
