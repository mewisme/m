package manifest

import (
	"encoding/json"
)

// Document is an on-disk package.json with preserved source bytes.
type Document struct {
	Path   string
	Source []byte

	Name                 string
	Version              string
	Private              bool
	Dependencies         map[string]string
	DevDependencies      map[string]string
	OptionalDependencies map[string]string
	PeerDependencies     map[string]string
	Overrides            map[string]json.RawMessage
	Scripts              map[string]string
	Engines              map[string]string
	PackageManager       string
	Workspaces           json.RawMessage // array or {"packages":[...]}
	Catalog              json.RawMessage // catalog entry map
	Bin                  json.RawMessage // string or object
}
