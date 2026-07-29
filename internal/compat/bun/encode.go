package bun

import (
	"bytes"
	"encoding/json"
	"sort"
)

// Encode serializes a bun lock document to JSONC-compatible bytes.
func Encode(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, nil
	}
	payload := encodePayload(doc)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return out, nil
}

func encodePayload(doc *Document) map[string]any {
	out := map[string]any{
		"lockfileVersion": doc.LockfileVersion,
	}
	if doc.ConfigVersion != 0 {
		out["configVersion"] = doc.ConfigVersion
	}
	if len(doc.Workspaces) > 0 {
		ws := map[string]any{}
		for _, path := range sortedWorkspacePathKeys(doc.Workspaces) {
			ws[path] = encodeWorkspace(doc.Workspaces[path])
		}
		out["workspaces"] = ws
	}
	if len(doc.Packages) > 0 {
		pkgs := map[string]any{}
		for _, name := range sortedStringKeys(doc.Packages) {
			pkgs[name] = encodePackageArray(doc.Packages[name])
		}
		out["packages"] = pkgs
	}
	for _, k := range sortedExtensionKeys(doc.Extensions) {
		var v any
		_ = json.Unmarshal(doc.Extensions[k], &v)
		out[k] = v
	}
	return out
}

func encodeWorkspace(entry WorkspaceEntry) map[string]any {
	out := map[string]any{}
	if entry.Name != "" {
		out["name"] = entry.Name
	}
	encodeStringMap(out, "dependencies", entry.Dependencies)
	encodeStringMap(out, "devDependencies", entry.DevDependencies)
	encodeStringMap(out, "optionalDependencies", entry.OptionalDependencies)
	for _, k := range sortedExtensionKeys(entry.Extra) {
		var v any
		_ = json.Unmarshal(entry.Extra[k], &v)
		out[k] = v
	}
	return out
}

func encodePackageArray(arr PackageArray) []any {
	out := make([]any, 0, len(arr))
	for _, raw := range arr {
		var v any
		_ = json.Unmarshal(raw, &v)
		out = append(out, v)
	}
	return out
}

func encodeStringMap(out map[string]any, key string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	keys := sortedStringKeys(m)
	encoded := make(map[string]string, len(m))
	for _, k := range keys {
		encoded[k] = m[k]
	}
	out[key] = encoded
}

func sortedWorkspacePathKeys(m map[string]WorkspaceEntry) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedStringKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedExtensionKeys(m map[string]json.RawMessage) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
