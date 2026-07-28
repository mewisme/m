package lockfile

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

// ImporterSpecifierDiff records importer dependency specifier changes.
type ImporterSpecifierDiff struct {
	Importer string `json:"importer"`
	Name     string `json:"name"`
	Kind     string `json:"kind,omitempty"`
	Before   string `json:"before,omitempty"`
	After    string `json:"after,omitempty"`
}

// GraphDiff is a semantic diff between two canonical graphs.
type GraphDiff struct {
	PackagesAdded   []string                `json:"packagesAdded,omitempty"`
	PackagesRemoved []string                `json:"packagesRemoved,omitempty"`
	EdgesAdded      []graph.Edge            `json:"edgesAdded,omitempty"`
	EdgesRemoved    []graph.Edge            `json:"edgesRemoved,omitempty"`
	Specifiers      []ImporterSpecifierDiff `json:"specifiers,omitempty"`
}

// GraphsEqual reports semantic equality after validation normalization.
func GraphsEqual(a, b *graph.Graph) (bool, error) {
	if a == nil || b == nil {
		return a == b, nil
	}
	ac, err := cloneGraph(a)
	if err != nil {
		return false, err
	}
	bc, err := cloneGraph(b)
	if err != nil {
		return false, err
	}
	aj, err := graph.EncodeJSON(ac)
	if err != nil {
		return false, err
	}
	bj, err := graph.EncodeJSON(bc)
	if err != nil {
		return false, err
	}
	return bytes.Equal(aj, bj), nil
}

// DiffGraphs compares two graphs and returns a stable semantic diff.
func DiffGraphs(a, b *graph.Graph) (*GraphDiff, error) {
	ac, err := cloneGraph(a)
	if err != nil {
		return nil, err
	}
	bc, err := cloneGraph(b)
	if err != nil {
		return nil, err
	}

	diff := &GraphDiff{}

	aPkgs := packageKeys(ac)
	bPkgs := packageKeys(bc)
	diff.PackagesAdded = sortedDiff(aPkgs, bPkgs)
	diff.PackagesRemoved = sortedDiff(bPkgs, aPkgs)

	aEdges := edgeKeys(ac.Edges)
	bEdges := edgeKeys(bc.Edges)
	for k, e := range bEdges {
		if _, ok := aEdges[k]; !ok {
			diff.EdgesAdded = append(diff.EdgesAdded, e)
		}
	}
	for k, e := range aEdges {
		if _, ok := bEdges[k]; !ok {
			diff.EdgesRemoved = append(diff.EdgesRemoved, e)
		}
	}
	sortEdges(diff.EdgesAdded)
	sortEdges(diff.EdgesRemoved)

	diff.Specifiers = diffImporterSpecifiers(ac, bc)
	sort.SliceStable(diff.Specifiers, func(i, j int) bool {
		a, b := diff.Specifiers[i], diff.Specifiers[j]
		if a.Importer != b.Importer {
			return a.Importer < b.Importer
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Kind < b.Kind
	})

	return diff, nil
}

// EncodeDiffJSON encodes a graph diff for CLI output.
func EncodeDiffJSON(d *GraphDiff) ([]byte, error) {
	if d == nil {
		return nil, apperr.New(apperr.Lockfile, "lock.diff.encode", "diff", "nil diff")
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "lock.diff.encode", "diff", err)
	}
	return buf.Bytes(), nil
}

func cloneGraph(g *graph.Graph) (*graph.Graph, error) {
	data, err := graph.EncodeJSON(g)
	if err != nil {
		return nil, err
	}
	return graph.DecodeJSON(data)
}

func packageKeys(g *graph.Graph) map[string]struct{} {
	out := make(map[string]struct{}, len(g.Packages))
	for _, p := range g.Packages {
		out[p.ID.Key()] = struct{}{}
	}
	return out
}

func edgeKeys(edges []graph.Edge) map[string]graph.Edge {
	out := make(map[string]graph.Edge, len(edges))
	for _, e := range edges {
		out[edgeKey(e)] = e
	}
	return out
}

func edgeKey(e graph.Edge) string {
	return e.From + "\x00" + e.Name + "\x00" + e.To + "\x00" + string(e.Kind) + "\x00" + e.Range
}

func sortedDiff(have, want map[string]struct{}) []string {
	var out []string
	for k := range want {
		if _, ok := have[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func sortEdges(edges []graph.Edge) {
	sort.SliceStable(edges, func(i, j int) bool {
		return edgeKey(edges[i]) < edgeKey(edges[j])
	})
}

func diffImporterSpecifiers(a, b *graph.Graph) []ImporterSpecifierDiff {
	aSpecs := importerSpecifiers(a)
	bSpecs := importerSpecifiers(b)
	var out []ImporterSpecifierDiff
	for key, after := range bSpecs {
		before, ok := aSpecs[key]
		if !ok {
			out = append(out, ImporterSpecifierDiff{
				Importer: key.importer,
				Name:     key.name,
				Kind:     key.kind,
				After:    after,
			})
			continue
		}
		if before != after {
			out = append(out, ImporterSpecifierDiff{
				Importer: key.importer,
				Name:     key.name,
				Kind:     key.kind,
				Before:   before,
				After:    after,
			})
		}
	}
	for key, before := range aSpecs {
		if _, ok := bSpecs[key]; !ok {
			out = append(out, ImporterSpecifierDiff{
				Importer: key.importer,
				Name:     key.name,
				Kind:     key.kind,
				Before:   before,
			})
		}
	}
	return out
}

type specKey struct {
	importer, name, kind string
}

func importerSpecifiers(g *graph.Graph) map[specKey]string {
	out := make(map[specKey]string)
	for _, e := range g.Edges {
		if e.Range == "" {
			continue
		}
		// Importer edges use importer id as From and carry the manifest specifier in Range.
		for _, im := range g.Importers {
			if string(im.ID) == e.From {
				out[specKey{importer: e.From, name: e.Name, kind: string(e.Kind)}] = e.Range
				break
			}
		}
	}
	return out
}
