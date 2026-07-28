package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"unicode"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// SetField sets a top-level JSON string field, preserving surrounding source layout.
func (d *Document) SetField(key, value string) error {
	if d == nil {
		return apperr.New(apperr.Manifest, "manifest.edit", key, "nil document")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return apperr.Wrap(apperr.Manifest, "manifest.edit", key, err)
	}
	if err := d.spliceTopLevel(key, raw); err != nil {
		return err
	}
	switch key {
	case "name":
		d.Name = value
	case "version":
		d.Version = value
	case "packageManager":
		d.PackageManager = value
	}
	return nil
}

// SetDependency sets name→range inside a top-level dependency object field.
func (d *Document) SetDependency(field, name, rng string) error {
	if d == nil {
		return apperr.New(apperr.Manifest, "manifest.edit", field, "nil document")
	}
	var m map[string]string
	switch field {
	case "dependencies":
		m = cloneMap(d.Dependencies)
	case "devDependencies":
		m = cloneMap(d.DevDependencies)
	case "optionalDependencies":
		m = cloneMap(d.OptionalDependencies)
	case "peerDependencies":
		m = cloneMap(d.PeerDependencies)
	default:
		return apperr.New(apperr.Manifest, "manifest.edit", field, "not a dependency field")
	}
	if m == nil {
		m = map[string]string{}
	}
	m[name] = rng
	raw, err := json.Marshal(m)
	if err != nil {
		return apperr.Wrap(apperr.Manifest, "manifest.edit", field, err)
	}
	if err := d.spliceTopLevel(field, raw); err != nil {
		return err
	}
	switch field {
	case "dependencies":
		d.Dependencies = m
	case "devDependencies":
		d.DevDependencies = m
	case "optionalDependencies":
		d.OptionalDependencies = m
	case "peerDependencies":
		d.PeerDependencies = m
	}
	return nil
}

// RemoveDependency deletes name from a top-level dependency object field.
func (d *Document) RemoveDependency(field, name string) error {
	if d == nil {
		return apperr.New(apperr.Manifest, "manifest.edit", field, "nil document")
	}
	var m map[string]string
	switch field {
	case "dependencies":
		m = cloneMap(d.Dependencies)
	case "devDependencies":
		m = cloneMap(d.DevDependencies)
	case "optionalDependencies":
		m = cloneMap(d.OptionalDependencies)
	case "peerDependencies":
		m = cloneMap(d.PeerDependencies)
	default:
		return apperr.New(apperr.Manifest, "manifest.edit", field, "not a dependency field")
	}
	if m == nil {
		return apperr.New(apperr.NotFound, "manifest.edit", name, "dependency not found")
	}
	if _, ok := m[name]; !ok {
		return apperr.New(apperr.NotFound, "manifest.edit", name, "dependency not found")
	}
	delete(m, name)
	raw, err := json.Marshal(m)
	if err != nil {
		return apperr.Wrap(apperr.Manifest, "manifest.edit", field, err)
	}
	if err := d.spliceTopLevel(field, raw); err != nil {
		return err
	}
	switch field {
	case "dependencies":
		d.Dependencies = m
	case "devDependencies":
		d.DevDependencies = m
	case "optionalDependencies":
		d.OptionalDependencies = m
	case "peerDependencies":
		d.PeerDependencies = m
	}
	return nil
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Write atomically replaces the document path (or path argument if non-empty).
func (d *Document) Write(path string) error {
	if d == nil {
		return apperr.New(apperr.Manifest, "manifest.write", "", "nil document")
	}
	if path == "" {
		path = d.Path
	}
	if path == "" {
		return apperr.New(apperr.Manifest, "manifest.write", "", "empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return apperr.Wrap(apperr.IO, "manifest.write", path, err)
	}
	if err := fsx.PublishFileDurable(abs, d.Source, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "manifest.write", abs, err)
	}
	d.Path = abs
	Invalidate(filepath.Dir(abs))
	return nil
}

func (d *Document) spliceTopLevel(key string, value []byte) error {
	start, end, found, err := findTopLevelValueSpan(d.Source, key)
	if err != nil {
		return err
	}
	if found {
		d.Source = concat3(d.Source[:start], value, d.Source[end:])
		return nil
	}
	return d.insertTopLevel(key, value)
}

func concat3(a, b, c []byte) []byte {
	out := make([]byte, 0, len(a)+len(b)+len(c))
	out = append(out, a...)
	out = append(out, b...)
	out = append(out, c...)
	return out
}

func findTopLevelValueSpan(src []byte, key string) (start, end int, found bool, err error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	tok, err := dec.Token()
	if err != nil {
		return 0, 0, false, apperr.Wrap(apperr.Manifest, "manifest.edit", key, err)
	}
	if tok != json.Delim('{') {
		return 0, 0, false, apperr.New(apperr.Manifest, "manifest.edit", key, "root must be object")
	}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return 0, 0, false, apperr.Wrap(apperr.Manifest, "manifest.edit", key, err)
		}
		k, ok := kt.(string)
		if !ok {
			return 0, 0, false, apperr.New(apperr.Manifest, "manifest.edit", key, "object key must be string")
		}
		valStart := int(dec.InputOffset())
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return 0, 0, false, apperr.Wrap(apperr.Manifest, "manifest.edit", key, err)
		}
		valEnd := int(dec.InputOffset())
		// Trim leading whitespace in the captured span so we only replace the value.
		for valStart < valEnd && unicode.IsSpace(rune(src[valStart])) {
			valStart++
		}
		// Decoder may leave trailing whitespace/newlines before next token inside offset;
		// RawMessage length is authoritative for the value itself.
		if len(raw) > 0 && valStart+len(raw) <= len(src) && bytes.Equal(src[valStart:valStart+len(raw)], raw) {
			valEnd = valStart + len(raw)
		} else {
			// Fallback: search raw within [valStart, valEnd]
			if i := bytes.Index(src[valStart:valEnd], raw); i >= 0 {
				valStart += i
				valEnd = valStart + len(raw)
			}
		}
		if k == key {
			return valStart, valEnd, true, nil
		}
	}
	return 0, 0, false, nil
}

func (d *Document) insertTopLevel(key string, value []byte) error {
	src := d.Source
	// Find last non-space before closing }
	i := len(src) - 1
	for i >= 0 && unicode.IsSpace(rune(src[i])) {
		i--
	}
	if i < 0 || src[i] != '}' {
		return apperr.New(apperr.Manifest, "manifest.edit", key, "cannot find closing brace")
	}
	closeIdx := i
	// Peek whether object has existing members
	j := closeIdx - 1
	for j >= 0 && unicode.IsSpace(rune(src[j])) {
		j--
	}
	empty := j >= 0 && src[j] == '{'
	indent := detectIndent(src)
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return apperr.Wrap(apperr.Manifest, "manifest.edit", key, err)
	}
	var insert []byte
	if empty {
		insert = []byte(fmt.Sprintf("\n%s%s: %s\n", indent, keyJSON, value))
	} else {
		insert = []byte(fmt.Sprintf(",\n%s%s: %s\n", indent, keyJSON, value))
		// If there is already a newline before }, avoid double newline mess:
		// place comma after last value: insert before closeIdx, after trimming trailing spaces on prior line.
	}
	d.Source = concat3(src[:closeIdx], insert, src[closeIdx:])
	return nil
}

func detectIndent(src []byte) string {
	// Prefer first indented key line.
	lines := bytes.Split(src, []byte("\n"))
	for _, line := range lines {
		trimmed := bytes.TrimLeftFunc(line, unicode.IsSpace)
		if len(trimmed) > 0 && trimmed[0] == '"' {
			n := len(line) - len(trimmed)
			if n > 0 {
				return string(line[:n])
			}
		}
	}
	return "  "
}
