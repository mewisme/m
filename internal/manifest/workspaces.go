package manifest

import (
	"encoding/json"

	"github.com/mewisme/mew/internal/apperr"
)

// WorkspacePatterns extracts workspace glob patterns from the workspaces field.
func (d *Document) WorkspacePatterns() ([]string, error) {
	if d == nil || len(d.Workspaces) == 0 {
		return nil, nil
	}
	return ParseWorkspacesField(d.Workspaces)
}

// ParseWorkspacesField accepts a JSON array or {"packages":[...]}.
func ParseWorkspacesField(raw json.RawMessage) ([]string, error) {
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var obj struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Packages, nil
	}
	return nil, apperr.New(apperr.Manifest, "manifest.workspaces", "workspaces",
		"workspaces must be a string array or {\"packages\":[...]}")
}
