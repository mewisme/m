package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
)

// Parse parses package.json bytes into a Document (strict JSON, duplicate keys rejected).
func Parse(data []byte) (*Document, error) {
	if err := checkDuplicateKeys(data); err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "manifest.parse", "package.json", err)
	}
	doc := &Document{
		Source: append([]byte(nil), data...),
	}
	if err := decodeString(raw, "name", &doc.Name); err != nil {
		return nil, err
	}
	if err := decodeString(raw, "version", &doc.Version); err != nil {
		return nil, err
	}
	if err := decodeBool(raw, "private", &doc.Private); err != nil {
		return nil, err
	}
	if err := decodeString(raw, "packageManager", &doc.PackageManager); err != nil {
		return nil, err
	}
	var err error
	if doc.Dependencies, err = decodeStringMap(raw, "dependencies"); err != nil {
		return nil, err
	}
	if doc.DevDependencies, err = decodeStringMap(raw, "devDependencies"); err != nil {
		return nil, err
	}
	if doc.OptionalDependencies, err = decodeStringMap(raw, "optionalDependencies"); err != nil {
		return nil, err
	}
	if doc.PeerDependencies, err = decodeStringMap(raw, "peerDependencies"); err != nil {
		return nil, err
	}
	if doc.Overrides, err = decodeRawMap(raw, "overrides"); err != nil {
		return nil, err
	}
	if doc.Scripts, err = decodeStringMap(raw, "scripts"); err != nil {
		return nil, err
	}
	if doc.Engines, err = decodeStringMap(raw, "engines"); err != nil {
		return nil, err
	}
	if v, ok := raw["workspaces"]; ok {
		doc.Workspaces = append(json.RawMessage(nil), v...)
	}
	if v, ok := raw["bin"]; ok {
		doc.Bin = append(json.RawMessage(nil), v...)
	}
	return doc, nil
}

// Load reads and parses path as package.json.
func Load(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.Wrap(apperr.NotFound, "manifest.load", path, err)
		}
		return nil, apperr.Wrap(apperr.IO, "manifest.load", path, err)
	}
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "manifest.load", path, err)
	}
	doc.Path = abs
	return doc, nil
}

func decodeString(raw map[string]json.RawMessage, key string, dst *string) error {
	v, ok := raw[key]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(v, dst); err != nil {
		return apperr.Wrap(apperr.Manifest, "manifest.parse", key, err)
	}
	return nil
}

func decodeBool(raw map[string]json.RawMessage, key string, dst *bool) error {
	v, ok := raw[key]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(v, dst); err != nil {
		return apperr.Wrap(apperr.Manifest, "manifest.parse", key, err)
	}
	return nil
}

func decodeRawMap(raw map[string]json.RawMessage, key string) (map[string]json.RawMessage, error) {
	v, ok := raw[key]
	if !ok {
		return nil, nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(v, &m); err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "manifest.parse", key, err)
	}
	return m, nil
}

func decodeStringMap(raw map[string]json.RawMessage, key string) (map[string]string, error) {
	v, ok := raw[key]
	if !ok {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(v, &m); err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "manifest.parse", key, err)
	}
	return m, nil
}

func checkDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return checkDupValue(dec, "package.json")
}

func checkDupValue(dec *json.Decoder, path string) error {
	tok, err := dec.Token()
	if err != nil {
		return apperr.Wrap(apperr.Manifest, "manifest.parse", path, err)
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			seen := map[string]struct{}{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return apperr.Wrap(apperr.Manifest, "manifest.parse", path, err)
				}
				key, ok := kt.(string)
				if !ok {
					return apperr.New(apperr.Manifest, "manifest.parse", path, "object key must be string")
				}
				if _, dup := seen[key]; dup {
					return apperr.New(apperr.Manifest, "manifest.parse", path,
						fmt.Sprintf("duplicate key %q", key))
				}
				seen[key] = struct{}{}
				child := path + "." + key
				if path == "package.json" {
					child = key
				}
				if err := checkDupValue(dec, child); err != nil {
					return err
				}
			}
			end, err := dec.Token()
			if err != nil {
				return apperr.Wrap(apperr.Manifest, "manifest.parse", path, err)
			}
			if end != json.Delim('}') {
				return apperr.New(apperr.Manifest, "manifest.parse", path, "expected }")
			}
		case '[':
			i := 0
			for dec.More() {
				if err := checkDupValue(dec, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
				i++
			}
			end, err := dec.Token()
			if err != nil {
				return apperr.Wrap(apperr.Manifest, "manifest.parse", path, err)
			}
			if end != json.Delim(']') {
				return apperr.New(apperr.Manifest, "manifest.parse", path, "expected ]")
			}
		default:
			return apperr.New(apperr.Manifest, "manifest.parse", path, fmt.Sprintf("unexpected delim %v", t))
		}
	case string, bool, float64, json.Number, nil:
		return nil
	default:
		return apperr.New(apperr.Manifest, "manifest.parse", path, fmt.Sprintf("unexpected token %T", tok))
	}
	return nil
}
