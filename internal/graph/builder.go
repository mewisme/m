package graph

// Builder accumulates graph parts for tests, then Validate freezes the result.
type Builder struct {
	importers []Importer
	packages  []Package
	edges     []Edge
}

// NewBuilder returns an empty graph builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Importer appends an importer.
func (b *Builder) Importer(id ImporterID, name string) *Builder {
	path := string(id)
	b.importers = append(b.importers, Importer{ID: id, Name: name, Path: path})
	return b
}

// Package appends a resolved package.
func (b *Builder) Package(id PackageID, integrity, tarball string) *Builder {
	id.Normalize()
	b.packages = append(b.packages, Package{ID: id, Integrity: integrity, TarballURL: tarball})
	return b
}

// Edge appends a dependency edge.
func (b *Builder) Edge(from, to string, kind DepKind, rng string) *Builder {
	b.edges = append(b.edges, Edge{From: from, To: to, Kind: kind, Range: rng})
	return b
}

// Build validates and returns an immutable graph.
func (b *Builder) Build() (*Graph, error) {
	g := &Graph{
		SchemaVersion: SchemaVersion,
		Importers:     append([]Importer(nil), b.importers...),
		Packages:      append([]Package(nil), b.packages...),
		Edges:         append([]Edge(nil), b.edges...),
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}
