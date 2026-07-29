package classic

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
)

const (
	FormatClassic = "yarn-classic"
)

// ponytail: O(n) scan caps; upgrade path is streaming parser with early abort.
const (
	maxLockBytes  = 32 << 20
	maxBlocks     = 100_000
	maxLineLen    = 16_384
	maxDescriptor = 4096
)

// Document is a parsed Yarn classic lockfile.
type Document struct {
	Blocks     map[string]Block
	Extensions lockfile.Extensions
	Detection  lockfile.Detection
}

// Block is one yarn.lock entry.
type Block struct {
	Descriptor   string
	Version      string
	Resolved     string
	Integrity    string
	Dependencies map[string]string
	Extra        map[string]string
}

// Decode parses a Yarn v1 lockfile.
func Decode(data []byte) (*Document, error) {
	if err := validateLockInput(data); err != nil {
		return nil, err
	}
	if bytes.Contains(data, []byte("__metadata:")) {
		return nil, lockfile.NewUnsupported("yarn.classic.decode", "yarn.lock", "berry lockfile requires berry adapter")
	}
	doc := &Document{
		Blocks:     map[string]Block{},
		Extensions: lockfile.Extensions{},
		Detection: lockfile.Detection{
			Format: FormatClassic, ProducerMajor: 1, Confidence: lockfile.DetectionCertain,
			Evidence: []string{"no __metadata"},
		},
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineLen)
	var currentDesc string
	inDeps := false
	for scanner.Scan() {
		line := scanner.Text()
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			desc := strings.TrimSpace(line)
			if strings.HasSuffix(desc, ":") {
				desc = strings.TrimSuffix(desc, ":")
				desc = strings.Trim(desc, `"`)
				if len(desc) > maxDescriptor {
					return nil, apperr.New(apperr.Lockfile, "yarn.classic.decode", desc, "descriptor too long")
				}
				if len(doc.Blocks) >= maxBlocks {
					return nil, apperr.New(apperr.Lockfile, "yarn.classic.decode", "yarn.lock",
						fmt.Sprintf("exceeds %d blocks", maxBlocks))
				}
				currentDesc = desc
				if _, ok := doc.Blocks[desc]; !ok {
					doc.Blocks[desc] = Block{
						Descriptor: desc, Dependencies: map[string]string{}, Extra: map[string]string{},
					}
				}
			}
			inDeps = false
			continue
		}
		if currentDesc == "" {
			continue
		}
		blk := doc.Blocks[currentDesc]
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		trim = strings.TrimSpace(line)
		if trim == "dependencies:" && indent <= 2 {
			inDeps = true
			continue
		}
		if inDeps && indent > 2 {
			key, val, ok := splitKeyVal(trim)
			if ok {
				blk.Dependencies[key] = unquote(val)
				doc.Blocks[currentDesc] = blk
			}
			continue
		}
		inDeps = false
		key, val, ok := splitKeyVal(trim)
		if !ok {
			continue
		}
		switch key {
		case "version":
			blk.Version = unquote(val)
		case "resolved":
			blk.Resolved = unquote(val)
		case "integrity":
			blk.Integrity = unquote(val)
		default:
			blk.Extra[key] = unquote(val)
		}
		doc.Blocks[currentDesc] = blk
	}
	if err := scanner.Err(); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "yarn.classic.decode", "yarn.lock", err)
	}
	return doc, nil
}

func validateLockInput(data []byte) error {
	if len(data) == 0 {
		return apperr.New(apperr.Lockfile, "yarn.classic.decode", "yarn.lock", "empty document")
	}
	if len(data) > maxLockBytes {
		return apperr.New(apperr.Lockfile, "yarn.classic.decode", "yarn.lock",
			fmt.Sprintf("lockfile exceeds %d byte limit", maxLockBytes))
	}
	return nil
}

func splitKeyVal(line string) (key, val string, ok bool) {
	i := strings.IndexAny(line, " \t")
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// DescriptorName extracts the package name from a yarn descriptor.
func DescriptorName(desc string) string {
	at := strings.Index(desc, "@")
	if at <= 0 {
		return desc
	}
	if strings.HasPrefix(desc, "@") {
		at2 := strings.Index(desc[1:], "@")
		if at2 >= 0 {
			return desc[:at2+1]
		}
	}
	return desc[:at]
}

// ParseVersion extracts name and version from a resolved block.
func ParseVersion(name, version string) (string, string) {
	version = strings.TrimSpace(version)
	if version == "" {
		return name, ""
	}
	if len(version) > 0 && (unicode.IsDigit(rune(version[0])) || version[0] == 'v') {
		return name, strings.TrimPrefix(version, "v")
	}
	return name, version
}
