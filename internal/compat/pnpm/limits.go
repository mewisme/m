package pnpm

import (
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
	"go.yaml.in/yaml/v3"
)

// ponytail: O(n) scan caps; upgrade path is streaming YAML with early abort.
const (
	maxLockBytes    = 32 << 20 // 32 MiB
	maxNestingDepth = 64
	maxMapEntries   = 100_000
)

func validateLockInput(data []byte) error {
	if len(data) == 0 {
		return apperr.New(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml", "empty document")
	}
	if len(data) > maxLockBytes {
		return apperr.New(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml",
			fmt.Sprintf("lockfile exceeds %d byte limit", maxLockBytes))
	}
	return nil
}

func validateYAMLStructure(node *yaml.Node, depth int, entries *int) error {
	if node == nil {
		return nil
	}
	if depth > maxNestingDepth {
		return apperr.New(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml",
			fmt.Sprintf("nesting exceeds depth %d", maxNestingDepth))
	}
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			return apperr.New(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml", "malformed mapping")
		}
		seen := make(map[string]struct{}, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if _, dup := seen[key]; dup {
				return apperr.New(apperr.Lockfile, "pnpm.decode", key, "duplicate key")
			}
			seen[key] = struct{}{}
			*entries++
			if *entries > maxMapEntries {
				return apperr.New(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml",
					fmt.Sprintf("exceeds %d map entries", maxMapEntries))
			}
			if err := validateYAMLStructure(node.Content[i+1], depth+1, entries); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := validateYAMLStructure(child, depth+1, entries); err != nil {
				return err
			}
		}
	case yaml.AliasNode:
		return apperr.New(apperr.Lockfile, "pnpm.decode", "pnpm-lock.yaml", "YAML aliases are not supported")
	}
	return nil
}
