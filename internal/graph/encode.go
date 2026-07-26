package graph

import (
	"bytes"
	"encoding/json"

	"github.com/mewisme/m/internal/apperr"
)

// EncodeJSON validates, then encodes with sorted collections and trailing newline.
func EncodeJSON(g *Graph) ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(g); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "graph.encode", "graph", err)
	}
	return buf.Bytes(), nil
}

// DecodeJSON unmarshals a graph document and validates it.
func DecodeJSON(data []byte) (*Graph, error) {
	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "graph.decode", "graph", err)
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}
