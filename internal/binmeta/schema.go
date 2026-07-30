package binmeta

// SchemaVersion is the current bins metadata format version.
const SchemaVersion = 1

// LayoutMode describes the node_modules layout that produced the metadata.
type LayoutMode string

const (
	LayoutHoisted  LayoutMode = "hoisted"
	LayoutIsolated LayoutMode = "isolated"
	LayoutPnP      LayoutMode = "pnp"
)

// Record is one command ownership entry in bins metadata.
type Record struct {
	DependencyName    string `json:"dependencyName"`
	ResolvedPackage   string `json:"resolvedPackageKey"`
	PackageDir        string `json:"packageDir"`
	DeclaredBin       string `json:"declaredBin"`
	MaterializedShim  string `json:"materializedShim"`
	OwnershipVerified bool   `json:"ownershipVerified"`
}

// Document is node_modules/.mew/bins.v1.json.
type Document struct {
	SchemaVersion    int        `json:"schemaVersion"`
	GenerationID     string     `json:"generationID"`
	ImporterIdentity string     `json:"importerIdentity"`
	LayoutMode       LayoutMode `json:"layoutMode"`
	Fingerprint      string     `json:"fingerprint"`
	Records          []Record   `json:"records"`
}
