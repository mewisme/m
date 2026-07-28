package lockfile

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// DetectPnpm inspects pnpm-lock.yaml bytes and returns producer generation hints.
// Do not trust lockfileVersion alone for 9.0-shaped locks.
func DetectPnpm(data []byte) (Detection, error) {
	return DetectPnpmWithMajor(data, 0)
}

// DetectPnpmWithMajor classifies pnpm locks; producerMajor disambiguates v9-shaped locks (9, 10, or 11).
func DetectPnpmWithMajor(data []byte, producerMajor int) (Detection, error) {
	root, err := parseYAMLRoot(data)
	if err != nil {
		return Detection{}, err
	}

	if hasKey(root, "importers") || hasKey(root, "snapshots") {
		return detectPnpmV9Family(root, producerMajor)
	}
	return detectPnpmV6(root)
}

func detectPnpmV6(root map[string]*yaml.Node) (Detection, error) {
	ver, ok := lockfileVersionString(root)
	if !ok {
		return Detection{}, NewUnsupported("lock.detect", "pnpm-lock.yaml", "missing lockfileVersion")
	}
	if !isV6Version(ver) {
		return Detection{}, NewUnsupported("lock.detect", "pnpm-lock.yaml",
			fmt.Sprintf("unsupported lockfileVersion %q", ver))
	}
	return Detection{
		Format:        "pnpm-v6",
		ProducerMajor: 0,
		Confidence:    DetectionCertain,
		Evidence:      []string{"lockfileVersion=" + ver, "layout=v6-flat"},
	}, nil
}

func detectPnpmV9Family(root map[string]*yaml.Node, producerMajor int) (Detection, error) {
	ver, ok := lockfileVersionString(root)
	if !ok || ver != "9.0" {
		return Detection{}, NewUnsupported("lock.detect", "pnpm-lock.yaml",
			fmt.Sprintf("unsupported 9.x-shaped lockfileVersion %q", ver))
	}

	evidence := []string{"lockfileVersion=9.0", "layout=importers-snapshots"}

	if hasV11Markers(root) {
		return Detection{
			Format:        "pnpm-v11",
			ProducerMajor: 11,
			Confidence:    DetectionCertain,
			Evidence:      append(evidence, "marker=v11-build-policy"),
		}, nil
	}
	if hasV10Markers(root) {
		return Detection{
			Format:        "pnpm-v10",
			ProducerMajor: 10,
			Confidence:    DetectionCertain,
			Evidence:      append(evidence, "marker=v10-patch-config"),
		}, nil
	}

	switch producerMajor {
	case 11:
		return Detection{
			Format:        "pnpm-v11",
			ProducerMajor: 11,
			Confidence:    DetectionInferred,
			ExplicitMajor: true,
			Evidence:      append(evidence, "explicit producerMajor=11"),
		}, nil
	case 10:
		return Detection{
			Format:        "pnpm-v10",
			ProducerMajor: 10,
			Confidence:    DetectionInferred,
			ExplicitMajor: true,
			Evidence:      append(evidence, "explicit producerMajor=10"),
		}, nil
	case 9:
		return Detection{
			Format:        "pnpm-v9",
			ProducerMajor: 9,
			Confidence:    DetectionInferred,
			ExplicitMajor: true,
			Evidence:      append(evidence, "explicit producerMajor=9"),
		}, nil
	}

	return Detection{
		Format:        "pnpm-v9",
		ProducerMajor: 9,
		Confidence:    DetectionInferred,
		Evidence:      append(evidence, "marker=ambiguous-v9-shaped"),
	}, nil
}

func hasV10Markers(root map[string]*yaml.Node) bool {
	return hasKey(root, "patchedDependencies") || hasKey(root, "configDependencies")
}

func hasV11Markers(root map[string]*yaml.Node) bool {
	settings, ok := root["settings"]
	if !ok || settings.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(settings.Content); i += 2 {
		key := settings.Content[i]
		if key.Value == "onlyBuiltDependencies" || key.Value == "ignoredBuiltDependencies" {
			return true
		}
	}
	return false
}

func isV6Version(ver string) bool {
	if strings.HasPrefix(ver, "5.") {
		return true
	}
	return ver == "6.0" || ver == "6"
}

func lockfileVersionString(root map[string]*yaml.Node) (string, bool) {
	node, ok := root["lockfileVersion"]
	if !ok || node.Kind != yaml.ScalarNode {
		return "", false
	}
	return strings.TrimSpace(node.Value), true
}

func hasKey(root map[string]*yaml.Node, key string) bool {
	_, ok := root[key]
	return ok
}

func parseYAMLRoot(data []byte) (map[string]*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, NewUnsupported("lock.detect", "pnpm-lock.yaml", err.Error())
	}
	if len(doc.Content) == 0 {
		return nil, NewUnsupported("lock.detect", "pnpm-lock.yaml", "empty document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, NewUnsupported("lock.detect", "pnpm-lock.yaml", "expected mapping root")
	}
	out := make(map[string]*yaml.Node, len(root.Content)/2)
	for i := 0; i < len(root.Content); i += 2 {
		key := root.Content[i]
		val := root.Content[i+1]
		out[key.Value] = val
	}
	return out, nil
}
