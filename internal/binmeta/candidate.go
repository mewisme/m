package binmeta

// BinCandidate is one resolved local binary visible from an importer.
type BinCandidate struct {
	Command           string
	DependencyName    string
	ResolvedPackage   string
	PackageDir        string
	ShimPath          string
	TargetPath        string
	OwnershipVerified bool
}
