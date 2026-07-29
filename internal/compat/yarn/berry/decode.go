package berry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/lockfile"
	"go.yaml.in/yaml/v3"
)

const (
	FormatBerryNM  = "yarn-berry-nm"
	FormatBerryPnP = "yarn-berry-pnp"
	ExtLinkerKey   = "mew.yarn/linker"
)

// ponytail: O(n) scan caps; upgrade path is streaming YAML with early abort.
const (
	maxLockBytes  = 32 << 20
	maxMapEntries = 100_000
)

// Document is a parsed Yarn Berry lockfile.
type Document struct {
	Metadata   Metadata
	Blocks     map[string]Block
	Extensions lockfile.Extensions
	Detection  lockfile.Detection
	Linker     string
}

// Metadata is the __metadata block.
type Metadata struct {
	Version  int
	CacheKey string
}

// Block is one berry lock entry.
type Block struct {
	Key          string
	Version      string
	Resolution   string
	Checksum     string
	LanguageName string
	LinkType     string
	Dependencies map[string]string
	Extra        map[string]string
}

// IsBerry reports whether lock bytes look like Yarn Berry format.
func IsBerry(data []byte) bool {
	trim := strings.TrimSpace(string(data))
	return strings.Contains(trim, "__metadata:") || strings.Contains(trim, "\"__metadata\"")
}

// HasPnPArtifact reports whether .pnp.cjs exists in the project root.
func HasPnPArtifact(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".pnp.cjs"))
	return err == nil
}

// Decode parses a Yarn Berry yarn.lock YAML document.
func Decode(data []byte) (*Document, error) {
	if len(data) == 0 {
		return nil, lockfile.NewUnsupported("yarn.berry.decode", "yarn.lock", "empty document")
	}
	if len(data) > maxLockBytes {
		return nil, lockfile.NewUnsupported("yarn.berry.decode", "yarn.lock",
			fmt.Sprintf("lockfile exceeds %d byte limit", maxLockBytes))
	}
	if !IsBerry(data) {
		return nil, lockfile.NewUnsupported("yarn.berry.decode", "yarn.lock", "missing __metadata (not a berry lockfile)")
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, lockfile.NewUnsupported("yarn.berry.decode", "yarn.lock", err.Error())
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, lockfile.NewUnsupported("yarn.berry.decode", "yarn.lock", "empty yaml document")
	}
	mapNode := root.Content[0]
	if mapNode.Kind != yaml.MappingNode {
		return nil, lockfile.NewUnsupported("yarn.berry.decode", "yarn.lock", "expected mapping root")
	}
	doc := &Document{
		Blocks:     map[string]Block{},
		Extensions: lockfile.Extensions{},
		Linker:     "node-modules",
	}
	for i := 0; i < len(mapNode.Content); i += 2 {
		keyNode := mapNode.Content[i]
		valNode := mapNode.Content[i+1]
		key := nodeScalar(keyNode)
		if key == "__metadata" {
			doc.Metadata = parseMetadata(valNode)
			continue
		}
		blk := parseBlock(key, valNode)
		doc.Blocks[key] = blk
	}
	if len(doc.Blocks) > maxMapEntries {
		return nil, lockfile.NewUnsupported("yarn.berry.decode", "yarn.lock",
			fmt.Sprintf("exceeds %d blocks", maxMapEntries))
	}
	doc.Detection = lockfile.Detection{
		Format: FormatBerryNM, ProducerMajor: 4, Confidence: lockfile.DetectionCertain,
		Evidence: []string{"__metadata"},
	}
	return doc, nil
}

func parseMetadata(node *yaml.Node) Metadata {
	var meta Metadata
	if node == nil || node.Kind != yaml.MappingNode {
		return meta
	}
	for i := 0; i < len(node.Content); i += 2 {
		switch nodeScalar(node.Content[i]) {
		case "version":
			_ = node.Content[i+1].Decode(&meta.Version)
		case "cacheKey":
			meta.CacheKey = nodeScalar(node.Content[i+1])
		}
	}
	return meta
}

func parseBlock(key string, node *yaml.Node) Block {
	blk := Block{Key: key, Dependencies: map[string]string{}, Extra: map[string]string{}}
	if node == nil || node.Kind != yaml.MappingNode {
		return blk
	}
	for i := 0; i < len(node.Content); i += 2 {
		field := nodeScalar(node.Content[i])
		val := node.Content[i+1]
		switch field {
		case "version":
			blk.Version = nodeScalar(val)
		case "resolution":
			blk.Resolution = nodeScalar(val)
		case "checksum":
			blk.Checksum = nodeScalar(val)
		case "languageName":
			blk.LanguageName = nodeScalar(val)
		case "linkType":
			blk.LinkType = nodeScalar(val)
		case "dependencies":
			if val.Kind == yaml.MappingNode {
				for j := 0; j < len(val.Content); j += 2 {
					blk.Dependencies[nodeScalar(val.Content[j])] = nodeScalar(val.Content[j+1])
				}
			}
		default:
			blk.Extra[field] = nodeScalar(val)
		}
	}
	return blk
}

func nodeScalar(n *yaml.Node) string {
	if n == nil {
		return ""
	}
	return strings.TrimSpace(n.Value)
}
