package binmeta

import "sort"

// SortRecords orders records deterministically: dependencyName, resolvedPackage, packageDir.
func SortRecords(recs []Record) {
	sort.Slice(recs, func(i, j int) bool {
		a, b := recs[i], recs[j]
		if a.DependencyName != b.DependencyName {
			return a.DependencyName < b.DependencyName
		}
		if a.ResolvedPackage != b.ResolvedPackage {
			return a.ResolvedPackage < b.ResolvedPackage
		}
		return a.PackageDir < b.PackageDir
	})
}

// SortCandidates orders bin candidates with the same key ordering.
func SortCandidates(cands []BinCandidate) {
	sort.Slice(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.DependencyName != b.DependencyName {
			return a.DependencyName < b.DependencyName
		}
		if a.ResolvedPackage != b.ResolvedPackage {
			return a.ResolvedPackage < b.ResolvedPackage
		}
		return a.PackageDir < b.PackageDir
	})
}
