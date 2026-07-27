package resolver

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/semver"
)

// PeerSearchStep records one environment searched during peer provider lookup.
type PeerSearchStep struct {
	Environment string               `json:"environment"`
	Candidates  []string             `json:"candidates,omitempty"`
	Rejected    []CandidateRejection `json:"rejected,omitempty"`
}

// CandidateRejection records a version considered but rejected for a peer range.
type CandidateRejection struct {
	Version string `json:"version,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// ConflictNode is one node in a peer conflict explanation tree.
type ConflictNode struct {
	Constraint        string               `json:"constraint,omitempty"`
	RequiringPackage  string               `json:"requiringPackage,omitempty"`
	Importer          string               `json:"importer,omitempty"`
	SearchPath        []string             `json:"searchPath,omitempty"`
	Candidates        []string             `json:"candidates,omitempty"`
	Rejected          []CandidateRejection `json:"rejected,omitempty"`
	AutoInstallPolicy bool                 `json:"autoInstallPolicy,omitempty"`
	Optional          bool                 `json:"optional,omitempty"`
	Remediation       string               `json:"remediation,omitempty"`
	Children          []ConflictNode       `json:"children,omitempty"`
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
	root := ConflictNode{
		Constraint:        conf.Peer + "@" + conf.Range,
		RequiringPackage:  conf.Package,
		Importer:          conf.Importer,
		SearchPath:        append([]string(nil), conf.SearchPath...),
		AutoInstallPolicy: conf.AutoInstallPolicy,
		Optional:          conf.Optional,
		Remediation:       peerRemediation(conf),
	}
	if conf.Incompatible && conf.FoundVersion != "" {
		root.Rejected = []CandidateRejection{{Version: conf.FoundVersion, Reason: "range mismatch (nearest provider)"}}
	}
	root.Children = conflictChildrenFromSteps(conf)
	if len(root.Children) == 0 {
		candidates, rejected := peerCandidateAnalysis(prior, conf.Peer, conf.Range)
		root.Candidates = candidates
		root.Rejected = rejected
	}
	return ConflictTree{Peer: conf.Peer, Root: root}
}

func conflictChildrenFromSteps(conf PeerConflict) []ConflictNode {
	if len(conf.SearchSteps) == 0 {
		return nil
	}
	children := make([]ConflictNode, 0, len(conf.SearchPath))
	for i, env := range conf.SearchPath {
		ancestry := ConflictNode{
			Constraint: "ancestry: " + env,
			Importer:   env,
		}
		if i < len(conf.SearchSteps) {
			step := conf.SearchSteps[i]
			envNode := ConflictNode{
				Constraint:        "environment: " + step.Environment,
				Importer:          step.Environment,
				Candidates:        append([]string(nil), step.Candidates...),
				Rejected:          append([]CandidateRejection(nil), step.Rejected...),
				Optional:          conf.Optional,
				AutoInstallPolicy: conf.AutoInstallPolicy,
			}
			if conf.StrictPeers {
				envNode.Constraint += " (strict peers)"
			}
			ancestry.Children = []ConflictNode{envNode}
		}
		children = append(children, ancestry)
	}
	for _, step := range conf.SearchSteps[len(conf.SearchPath):] {
		children = append(children, ConflictNode{
			Constraint:        "environment: " + step.Environment,
			Importer:          step.Environment,
			Candidates:        append([]string(nil), step.Candidates...),
			Rejected:          append([]CandidateRejection(nil), step.Rejected...),
			Optional:          conf.Optional,
			AutoInstallPolicy: conf.AutoInstallPolicy,
		})
	}
	return children
}

func peerRemediation(conf PeerConflict) string {
	if conf.Optional {
		return fmt.Sprintf("mark %s optional in peerDependenciesMeta or install a compatible provider", conf.Peer)
	}
	if conf.AutoInstallPolicy {
		return fmt.Sprintf("install %s@%s or relax auto-install peer policy", conf.Peer, conf.Range)
	}
	return fmt.Sprintf("install %s@%s in the dependency tree or enable resolve.autoInstallPeers", conf.Peer, conf.Range)
}

func peerCandidateAnalysis(g *graph.Graph, peerName, rng string) (candidates []string, rejected []CandidateRejection) {
	if g == nil {
		return []string{"(none resolved)"}, nil
	}
	type ver struct {
		raw string
	}
	var versions []ver
	for _, p := range g.Packages {
		if p.ID.Name != peerName {
			continue
		}
		versions = append(versions, ver{raw: p.ID.Version})
	}
	sort.Slice(versions, func(i, j int) bool {
		cmp, err := semver.Compare(versions[i].raw, versions[j].raw)
		if err != nil {
			return versions[i].raw < versions[j].raw
		}
		return cmp < 0
	})
	for _, v := range versions {
		ok, err := semver.Satisfies(v.raw, rng)
		if err != nil {
			rejected = append(rejected, CandidateRejection{Version: v.raw, Reason: "invalid version"})
			continue
		}
		if ok {
			candidates = append(candidates, v.raw)
		} else {
			rejected = append(rejected, CandidateRejection{Version: v.raw, Reason: "range mismatch"})
		}
	}
	if len(candidates) == 0 && len(rejected) == 0 {
		candidates = []string{"(none resolved)"}
	}
	return candidates, rejected
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

// FormatConflictTree renders a human-readable conflict tree.
func FormatConflictTree(tree ConflictTree) string {
	var b strings.Builder
	fmt.Fprintf(&b, "peer %s\n", tree.Peer)
	formatConflictNode(&b, tree.Root, 0)
	return b.String()
}

func formatConflictNode(b *strings.Builder, n ConflictNode, depth int) {
	prefix := strings.Repeat("  ", depth)
	line := prefix + n.Constraint
	if n.RequiringPackage != "" {
		line += fmt.Sprintf(" required by %s", n.RequiringPackage)
	}
	if n.Importer != "" && n.Importer != n.RequiringPackage {
		line += fmt.Sprintf(" (importer %s)", n.Importer)
	}
	fmt.Fprintln(b, line)
	if len(n.SearchPath) > 0 {
		fmt.Fprintf(b, "%ssearch: %s\n", prefix, strings.Join(n.SearchPath, " → "))
	}
	if len(n.Candidates) > 0 {
		fmt.Fprintf(b, "%scandidates: %s\n", prefix, strings.Join(n.Candidates, ", "))
	}
	for _, r := range n.Rejected {
		fmt.Fprintf(b, "%srejected %s: %s\n", prefix, r.Version, r.Reason)
	}
	if n.Optional {
		fmt.Fprintf(b, "%soptional peer\n", prefix)
	}
	if n.AutoInstallPolicy {
		fmt.Fprintf(b, "%sauto-install peers enabled\n", prefix)
	}
	if n.Remediation != "" {
		fmt.Fprintf(b, "%sremediation: %s\n", prefix, n.Remediation)
	}
	for _, child := range n.Children {
		formatConflictNode(b, child, depth+1)
	}
}
