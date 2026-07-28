package pnpm

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
	"go.yaml.in/yaml/v3"
)

// Decode parses pnpm lock YAML into a Document.
func Decode(data []byte) (*Document, error) {
	if err := validateLockInput(data); err != nil {
		return nil, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml", err)
	}
	entries := 0
	if err := validateYAMLStructure(&root, 0, &entries); err != nil {
		return nil, err
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, apperr.New(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml", "empty document")
	}
	mapNode := root.Content[0]
	if mapNode.Kind != yaml.MappingNode {
		return nil, apperr.New(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml", "expected mapping")
	}
	if err := checkMappingKeys(mapNode, "pnpm-lock.yaml"); err != nil {
		return nil, err
	}

	doc := &Document{
		Settings:     map[string]any{},
		Importers:    map[string]ImporterSection{},
		Packages:     map[string]PackageEntry{},
		Snapshots:    map[string]map[string]any{},
		Dependencies: map[string]ImporterDep{},
		Extensions:   lockfile.Extensions{},
	}

	for i := 0; i < len(mapNode.Content); i += 2 {
		keyNode := mapNode.Content[i]
		valNode := mapNode.Content[i+1]
		key := keyNode.Value
		switch key {
		case "lockfileVersion":
			doc.LockfileVersion = strings.TrimSpace(valNode.Value)
		case "settings":
			doc.Settings = nodeToMap(valNode)
		case "importers":
			if err := decodeImporters(valNode, doc); err != nil {
				return nil, err
			}
		case "packages":
			if err := decodePackages(valNode, doc); err != nil {
				return nil, err
			}
		case "snapshots":
			doc.Snapshots = decodeSnapshotMap(valNode)
		case "dependencies":
			doc.Dependencies = decodeDepMap(valNode)
		default:
			raw, err := nodeToRawJSON(valNode)
			if err != nil {
				return nil, apperr.Wrap(apperr.Lockfile, "pnpm.decode", key, err)
			}
			doc.Extensions[key] = raw
		}
	}

	return doc, nil
}

func decodeImporters(node *yaml.Node, doc *Document) error {
	if node.Kind != yaml.MappingNode {
		return apperr.New(apperr.Lockfile, "pnpm.decode", "importers", "expected mapping")
	}
	for i := 0; i < len(node.Content); i += 2 {
		id := node.Content[i].Value
		secNode := node.Content[i+1]
		sec := ImporterSection{}
		if secNode.Kind != yaml.MappingNode {
			return apperr.New(apperr.Lockfile, "pnpm.decode", "importers", "expected mapping")
		}
		for j := 0; j < len(secNode.Content); j += 2 {
			depKind := secNode.Content[j].Value
			depNode := secNode.Content[j+1]
			switch depKind {
			case "dependencies":
				sec.Dependencies = decodeDepMap(depNode)
			case "devDependencies":
				sec.DevDependencies = decodeDepMap(depNode)
			case "optionalDependencies":
				sec.OptionalDependencies = decodeDepMap(depNode)
			case "dependenciesMeta":
				sec.DependenciesMeta = nodeToMap(depNode)
			case "publishDirectory":
				if depNode.Kind == yaml.ScalarNode {
					sec.PublishDirectory = depNode.Value
				}
			default:
				raw, err := nodeToRawJSON(depNode)
				if err != nil {
					return apperr.Wrap(apperr.Lockfile, "pnpm.decode", "importers."+id+"."+depKind, err)
				}
				if sec.Extra == nil {
					sec.Extra = map[string]json.RawMessage{}
				}
				sec.Extra[depKind] = raw
			}
		}
		doc.Importers[id] = sec
	}
	return nil
}

func decodePackages(node *yaml.Node, doc *Document) error {
	if node.Kind != yaml.MappingNode {
		return apperr.New(apperr.Lockfile, "pnpm.decode", "packages", "expected mapping")
	}
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		entryNode := node.Content[i+1]
		entry := PackageEntry{Extra: map[string]any{}}
		if entryNode.Kind != yaml.MappingNode {
			return apperr.New(apperr.Lockfile, "pnpm.decode", "packages", "expected mapping")
		}
		for j := 0; j < len(entryNode.Content); j += 2 {
			fk := entryNode.Content[j].Value
			fv := entryNode.Content[j+1]
			switch fk {
			case "resolution":
				entry.Resolution = nodeToMap(fv)
			case "engines":
				entry.Engines = nodeToMap(fv)
			case "dependencies":
				entry.Dependencies = decodeStringMap(fv)
			case "checksum":
				entry.Checksum = fv.Value
			case "buildPolicy":
				entry.BuildPolicy = nodeToAny(fv)
			default:
				entry.Extra[fk] = nodeToAny(fv)
			}
		}
		doc.Packages[key] = entry
	}
	return nil
}

func decodeDepMap(node *yaml.Node) map[string]ImporterDep {
	out := map[string]ImporterDep{}
	if node == nil || node.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i < len(node.Content); i += 2 {
		name := node.Content[i].Value
		depNode := node.Content[i+1]
		dep := ImporterDep{}
		if depNode.Kind == yaml.MappingNode {
			for j := 0; j < len(depNode.Content); j += 2 {
				switch depNode.Content[j].Value {
				case "specifier":
					dep.Specifier = depNode.Content[j+1].Value
				case "version":
					dep.Version = depNode.Content[j+1].Value
				}
			}
		}
		out[name] = dep
	}
	return out
}

func decodeStringMap(node *yaml.Node) map[string]string {
	out := map[string]string{}
	if node == nil || node.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i < len(node.Content); i += 2 {
		out[node.Content[i].Value] = node.Content[i+1].Value
	}
	return out
}

func decodeSnapshotMap(node *yaml.Node) map[string]map[string]any {
	out := map[string]map[string]any{}
	if node == nil || node.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i < len(node.Content); i += 2 {
		out[node.Content[i].Value] = nodeToMap(node.Content[i+1])
	}
	return out
}

func nodeToMap(node *yaml.Node) map[string]any {
	out := map[string]any{}
	if node == nil || node.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i < len(node.Content); i += 2 {
		out[node.Content[i].Value] = nodeToAny(node.Content[i+1])
	}
	return out
}

func nodeToAny(node *yaml.Node) any {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		return nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return scalarToAny(node)
	case yaml.MappingNode:
		return nodeToMap(node)
	case yaml.SequenceNode:
		items := make([]any, 0, len(node.Content))
		for _, c := range node.Content {
			items = append(items, nodeToAny(c))
		}
		return items
	default:
		return node.Value
	}
}

func nodeToRawJSON(node *yaml.Node) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(node); err != nil {
		return nil, err
	}
	_ = enc.Close()
	// yaml round-trip through generic map for JSON storage
	var v any
	if err := yaml.Unmarshal(buf.Bytes(), &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// IsV6Layout reports flat v5/v6 lockfiles without importers/snapshots.
func IsV6Layout(doc *Document) bool {
	if doc == nil {
		return false
	}
	ver := doc.LockfileVersion
	if strings.HasPrefix(ver, "5.") || ver == "6.0" || ver == "6" {
		return len(doc.Importers) == 0 && len(doc.Snapshots) == 0
	}
	return false
}
