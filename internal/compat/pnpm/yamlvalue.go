package pnpm

import (
	"strconv"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"go.yaml.in/yaml/v3"
)

// scalarToAny decodes YAML scalars with tag-aware typing while preserving quoted forms.
func scalarToAny(node *yaml.Node) any {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		return nil // aliases rejected at mapping parse
	}
	if node.Kind != yaml.ScalarNode {
		return node.Value
	}
	if node.Style&(yaml.SingleQuotedStyle|yaml.DoubleQuotedStyle|yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		return node.Value
	}
	tag := node.ShortTag()
	if tag == "" || tag == "!!str" {
		return node.Value
	}
	switch tag {
	case "!!bool":
		switch strings.ToLower(node.Value) {
		case "true":
			return true
		case "false":
			return false
		default:
			return node.Value
		}
	case "!!int":
		if i, err := strconv.ParseInt(node.Value, 10, 64); err == nil {
			return i
		}
		return node.Value
	case "!!float":
		if f, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return f
		}
		return node.Value
	case "!!null":
		return nil
	default:
		return node.Value
	}
}

func checkMappingKeys(node *yaml.Node, path string) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	seen := make(map[string]struct{}, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if keyNode.Kind == yaml.AliasNode || valNode.Kind == yaml.AliasNode {
			return apperr.New(apperr.Lockfile, "pnpm.decode", path, "YAML aliases are not supported")
		}
		key := keyNode.Value
		if _, dup := seen[key]; dup {
			return apperr.New(apperr.Lockfile, "pnpm.decode", path, "duplicate key "+key)
		}
		seen[key] = struct{}{}
		switch valNode.Kind {
		case yaml.MappingNode:
			if err := checkMappingKeys(valNode, path+"."+key); err != nil {
				return err
			}
		case yaml.SequenceNode:
			for _, item := range valNode.Content {
				if item.Kind == yaml.AliasNode {
					return apperr.New(apperr.Lockfile, "pnpm.decode", path, "YAML aliases are not supported")
				}
				if item.Kind == yaml.MappingNode {
					if err := checkMappingKeys(item, path+"."+key); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
