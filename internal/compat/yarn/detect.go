package yarn

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/compat/yarn/berry"
	"github.com/mewisme/mew/internal/compat/yarn/classic"
)

// Variant classifies a yarn.lock format.
type Variant string

const (
	VariantClassic  Variant = "classic"
	VariantBerryNM  Variant = "berry-nm"
	VariantBerryPnP Variant = "berry-pnp"
)

// DetectVariant inspects lock bytes and project root for yarn format.
func DetectVariant(lock []byte, root string) Variant {
	if berry.IsBerry(lock) {
		if HasPnPLinker(root) || berry.HasPnPArtifact(root) {
			return VariantBerryPnP
		}
		return VariantBerryNM
	}
	return VariantClassic
}

// HasPnPLinker reports whether .yarnrc.yml requests the PnP linker.
func HasPnPLinker(root string) bool {
	data, err := os.ReadFile(filepath.Join(root, ".yarnrc.yml"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nodeLinker:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "nodeLinker:"))
			return val == "pnp"
		}
	}
	return false
}

// Decode dispatches to classic or berry parsers.
func Decode(lock []byte, root string) (any, Variant, error) {
	switch DetectVariant(lock, root) {
	case VariantBerryNM, VariantBerryPnP:
		doc, err := berry.Decode(lock)
		if err != nil {
			return nil, "", err
		}
		return doc, DetectVariant(lock, root), nil
	default:
		doc, err := classic.Decode(lock)
		if err != nil {
			return nil, "", err
		}
		return doc, VariantClassic, nil
	}
}

// IsBerryLock reports whether bytes look like a Yarn Berry lockfile.
func IsBerryLock(lock []byte) bool {
	return berry.IsBerry(lock)
}

// TrimLeadingComment skips yarn classic autogen header lines.
func TrimLeadingComment(data []byte) []byte {
	lines := bytes.Split(data, []byte("\n"))
	start := 0
	for start < len(lines) {
		line := bytes.TrimSpace(lines[start])
		if len(line) == 0 || line[0] == '#' {
			start++
			continue
		}
		break
	}
	if start == 0 {
		return data
	}
	return bytes.Join(lines[start:], []byte("\n"))
}
