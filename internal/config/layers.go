package config

import (
	"errors"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// ErrNotSet reports that a key carries no value at the requested scope.
// It is a distinct condition from an unknown key, which is a Config error.
var ErrNotSet = errors.New("not set at scope")

// Layer is one retained configuration source with only the values that source
// itself declared. Layers are never flattened into each other, so provenance
// and shadowed values survive load and can be replayed by Explain.
type Layer struct {
	Source Source
	Path   string
	Values map[string]any
}

// layerOrder lists sources from lowest to highest precedence. A later layer
// shadows an earlier one for the same key.
var layerOrder = []Source{SourceDefaults, SourceGlobal, SourceProject, SourceEnv, SourceCLI}

// SourceForScope maps a writable scope onto the layer that backs it.
// ScopeEffective has no single backing layer and returns false.
func SourceForScope(scope Scope) (Source, bool) {
	switch scope {
	case ScopeUser:
		return SourceGlobal, true
	case ScopeProject:
		return SourceProject, true
	default:
		return "", false
	}
}

// newLayer allocates an empty layer for src.
func newLayer(src Source, path string) *Layer {
	return &Layer{Source: src, Path: path, Values: map[string]any{}}
}

// layer returns the retained layer for src, or nil when the load produced none.
func (e *Effective) layer(src Source) *Layer {
	if e == nil {
		return nil
	}
	for i := range e.Layers {
		if e.Layers[i].Source == src {
			return &e.Layers[i]
		}
	}
	return nil
}

// Layers returns the retained layers ordered lowest to highest precedence.
func (e *Effective) LayerList() []Layer {
	if e == nil {
		return nil
	}
	out := make([]Layer, 0, len(e.Layers))
	for _, src := range layerOrder {
		if l := e.layer(src); l != nil {
			out = append(out, *l)
		}
	}
	return out
}

// GetAtScope returns the value key carries in exactly one scope, ignoring every
// other layer. Missing keys return ErrNotSet so callers can tell "absent here"
// apart from "absent everywhere" and from "unknown key".
func GetAtScope(eff *Effective, scope Scope, key string) (Value, error) {
	if eff == nil {
		return Value{}, apperr.New(apperr.Config, "config.get", key, "nil config")
	}
	if scope == ScopeEffective {
		return Get(eff, key)
	}
	src, ok := SourceForScope(scope)
	if !ok {
		return Value{}, apperr.New(apperr.Config, "config.get", string(scope), "unknown scope")
	}
	canon := key
	if c := CanonicalKey(key); c != "" {
		canon = c
	} else if !strings.HasPrefix(key, "registries.") {
		// Unknown key is a different failure from "known but unset here";
		// reporting ErrNotSet would let a typo look like an empty scope.
		return Value{}, apperr.New(apperr.Config, "config.get", key, "unknown key")
	}
	l := eff.layer(src)
	if l == nil {
		return Value{}, ErrNotSet
	}
	raw, ok := l.Values[canon]
	if !ok {
		return Value{}, ErrNotSet
	}
	return Value{Raw: raw, Source: src, Path: l.Path}, nil
}

// ListAtScope returns the sorted entries a single scope declares. Unlike List
// it does not fall back to lower layers, so an empty result means the scope
// declares nothing.
func ListAtScope(eff *Effective, scope Scope) []Entry {
	if eff == nil {
		return nil
	}
	if scope == ScopeEffective {
		return List(eff)
	}
	src, ok := SourceForScope(scope)
	if !ok {
		return nil
	}
	l := eff.layer(src)
	if l == nil {
		return nil
	}
	keys := make([]string, 0, len(l.Values))
	for k := range l.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Entry, 0, len(keys))
	for _, k := range keys {
		out = append(out, Entry{
			Key:    k,
			Value:  formatRaw(l.Values[k]),
			Values: AllowedValues(k),
			Source: src,
			Path:   l.Path,
		})
	}
	return out
}

// GetEffective returns the winning value across all layers.
func GetEffective(eff *Effective, key string) (Value, error) {
	return Get(eff, key)
}

// ExplainEntry is one rung of a key's resolution chain.
type ExplainEntry struct {
	Source    Source
	Path      string
	Raw       any
	Effective bool // true for the layer that won
}

// Explain returns every layer that defines key, ordered lowest to highest
// precedence, with the winning layer marked. An unknown key is an error; a
// known key with no layer beyond defaults still returns its defaults rung.
func Explain(eff *Effective, key string) ([]ExplainEntry, error) {
	if eff == nil {
		return nil, apperr.New(apperr.Config, "config.explain", key, "nil config")
	}
	winner, err := Get(eff, key)
	if err != nil {
		return nil, err
	}
	canon := key
	if c := CanonicalKey(key); c != "" {
		canon = c
	}
	var out []ExplainEntry
	for _, src := range layerOrder {
		l := eff.layer(src)
		if l == nil {
			continue
		}
		raw, ok := l.Values[canon]
		if !ok {
			continue
		}
		out = append(out, ExplainEntry{Source: src, Path: l.Path, Raw: raw})
	}
	// The winner is the highest-precedence rung present; mark it there rather
	// than by comparing values, so two layers holding the same value do not
	// both claim to be effective.
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Source == winner.Source {
			out[i].Effective = true
			break
		}
	}
	return out, nil
}
