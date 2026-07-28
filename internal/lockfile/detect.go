package lockfile

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// DetectionExtensionKey stores adapter-recorded producer metadata in lock extensions.
const DetectionExtensionKey = "mew.lockfile/detection"

// ProjectHints carries package.json identity fields for detection.
type ProjectHints struct {
	PackageManager string
	DevEnginesPM   string
}

// DetectionConflictError reports disagreeing detection signals.
type DetectionConflictError struct {
	Signals []string
}

func (e *DetectionConflictError) Error() string {
	if e == nil || len(e.Signals) == 0 {
		return "lock detection conflict"
	}
	return "lock detection conflict: " + strings.Join(e.Signals, "; ")
}

type detectionCandidate struct {
	format string
	major  int
	source string
	cert   bool
}

// DetectPnpm inspects pnpm-lock.yaml bytes and returns producer generation hints.
// Do not trust lockfileVersion alone for 9.0-shaped locks.
func DetectPnpm(data []byte) (Detection, error) {
	return DetectPnpmWithMajor(data, 0)
}

// DetectPnpmWithMajor classifies pnpm locks; producerMajor disambiguates v9-shaped locks (9, 10, or 11).
func DetectPnpmWithMajor(data []byte, producerMajor int) (Detection, error) {
	return DetectPnpmWithContext(data, ProjectHints{}, producerMajor)
}

// DetectPnpmWithContext applies evidence order: packageManager, devEngines, explicit major,
// adapter extension metadata, policy structural evidence, else ambiguous.
func DetectPnpmWithContext(data []byte, hints ProjectHints, explicitMajor int) (Detection, error) {
	if PnpmValidateSupported != nil {
		if err := PnpmValidateSupported(data); err != nil {
			return Detection{}, err
		}
	}
	root, err := parseYAMLRoot(data)
	if err != nil {
		return Detection{}, err
	}

	if hasKey(root, "importers") || hasKey(root, "snapshots") {
		return detectPnpmV9Family(root, data, hints, explicitMajor)
	}
	return detectPnpmLegacyFlat(root)
}

func detectPnpmLegacyFlat(root map[string]*yaml.Node) (Detection, error) {
	ver, ok := lockfileVersionString(root)
	if !ok {
		return Detection{}, NewUnsupported("lock.detect", "pnpm-lock.yaml", "missing lockfileVersion")
	}
	if !isLegacyFlatVersion(ver) {
		return Detection{}, NewUnsupported("lock.detect", "pnpm-lock.yaml",
			fmt.Sprintf("unsupported lockfileVersion %q", ver))
	}
	legacy := NewPnpmLegacyUnsupported(ver, legacyLayoutForVersion(ver))
	return Detection{}, NewUnsupported("lock.detect", "pnpm-lock.yaml", legacy.Error())
}

func detectPnpmV9Family(root map[string]*yaml.Node, data []byte, hints ProjectHints, explicitMajor int) (Detection, error) {
	ver, ok := lockfileVersionString(root)
	if !ok || ver != "9.0" {
		if ok && isLegacyFlatVersion(ver) {
			legacy := NewPnpmLegacyUnsupported(ver, legacyLayoutForVersion(ver))
			return Detection{}, NewUnsupported("lock.detect", "pnpm-lock.yaml", legacy.Error())
		}
		return Detection{}, NewUnsupported("lock.detect", "pnpm-lock.yaml",
			fmt.Sprintf("unsupported 9.x-shaped lockfileVersion %q (only pnpm 9/10/11 with lockfileVersion 9.0 are supported)", ver))
	}

	evidence := []string{"lockfileVersion=9.0", "layout=importers-snapshots"}
	var candidates []detectionCandidate

	if major, ok := majorFromPackageManager(hints.PackageManager); ok {
		candidates = append(candidates, detectionCandidate{
			format: pnpmFormatForMajor(major), major: major, source: "packageManager", cert: true,
		})
	}
	if major, ok := majorFromPackageManager(hints.DevEnginesPM); ok {
		candidates = append(candidates, detectionCandidate{
			format: pnpmFormatForMajor(major), major: major, source: "devEngines.packageManager", cert: true,
		})
	}
	if explicitMajor >= 9 && explicitMajor <= 11 {
		candidates = append(candidates, detectionCandidate{
			format: pnpmFormatForMajor(explicitMajor), major: explicitMajor, source: "explicit producerMajor", cert: false,
		})
	}
	if major, ok := majorFromExtension(root); ok {
		candidates = append(candidates, detectionCandidate{
			format: pnpmFormatForMajor(major), major: major, source: DetectionExtensionKey, cert: true,
		})
	}
	if det, ok := pnpmStructureDetect(data); ok {
		candidates = append(candidates, detectionCandidate{
			format: det.Format, major: det.ProducerMajor, source: "structural evidence", cert: det.Confidence == DetectionCertain,
		})
	}

	if len(candidates) == 0 {
		return Detection{
			Format: "pnpm-v9", ProducerMajor: 9, Confidence: DetectionInferred,
			Evidence: append(evidence, "marker=ambiguous-v9-shaped"),
		}, nil
	}

	winner := candidates[0]
	for _, c := range candidates[1:] {
		if c.major != winner.major {
			signals := make([]string, 0, len(candidates))
			for _, x := range candidates {
				signals = append(signals, fmt.Sprintf("%s→pnpm-%d", x.source, x.major))
			}
			return Detection{}, &DetectionConflictError{Signals: signals}
		}
	}

	conf := DetectionInferred
	if winner.cert {
		conf = DetectionCertain
	}
	explicit := explicitMajor >= 9 && explicitMajor <= 11 && winner.source == "explicit producerMajor"
	return Detection{
		Format:        winner.format,
		ProducerMajor: winner.major,
		Confidence:    conf,
		ExplicitMajor: explicit,
		Evidence:      append(evidence, "source="+winner.source),
	}, nil
}

func pnpmStructureDetect(data []byte) (Detection, bool) {
	if PnpmStructureDetect == nil {
		return Detection{}, false
	}
	return PnpmStructureDetect(data)
}

func pnpmFormatForMajor(major int) string {
	switch major {
	case 10:
		return "pnpm-v10"
	case 11:
		return "pnpm-v11"
	default:
		return "pnpm-v9"
	}
}

func majorFromPackageManager(pm string) (int, bool) {
	pm = strings.TrimSpace(pm)
	if pm == "" {
		return 0, false
	}
	name := pm
	if i := strings.IndexByte(pm, '@'); i >= 0 {
		name = pm[:i]
	}
	if name != "pnpm" {
		return 0, false
	}
	if i := strings.LastIndexByte(pm, '@'); i >= 0 && i < len(pm)-1 {
		ver := pm[i+1:]
		if dot := strings.IndexByte(ver, '.'); dot > 0 {
			ver = ver[:dot]
		}
		if n, err := strconv.Atoi(ver); err == nil {
			if n >= 9 && n <= 11 {
				return n, true
			}
			return 0, false
		}
	}
	return 9, true
}

func majorFromExtension(root map[string]*yaml.Node) (int, bool) {
	node, ok := root[DetectionExtensionKey]
	if !ok {
		return 0, false
	}
	var meta struct {
		ProducerMajor int `json:"producerMajor"`
	}
	raw, err := yamlNodeToJSON(node)
	if err != nil {
		return 0, false
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return 0, false
	}
	if meta.ProducerMajor >= 9 && meta.ProducerMajor <= 11 {
		return meta.ProducerMajor, true
	}
	return 0, false
}

func yamlNodeToJSON(node *yaml.Node) ([]byte, error) {
	var v any
	if err := node.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func isLegacyFlatVersion(ver string) bool {
	if strings.HasPrefix(ver, "5.") {
		return true
	}
	switch ver {
	case "6", "6.0", "7", "7.0", "8", "8.0":
		return true
	}
	if strings.HasPrefix(ver, "7.") || strings.HasPrefix(ver, "8.") {
		return true
	}
	return false
}

func legacyLayoutForVersion(ver string) string {
	if strings.HasPrefix(ver, "5.") {
		return "v5-flat"
	}
	if ver == "6" || ver == "6.0" || strings.HasPrefix(ver, "6.") {
		return "v6-flat"
	}
	if strings.HasPrefix(ver, "7.") || ver == "7" || ver == "7.0" {
		return "v7-flat"
	}
	return "v8-flat"
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
