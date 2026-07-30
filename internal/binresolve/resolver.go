package binresolve

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
)

// Resolve finds the nearest-wins local binary for one importer context.
func Resolve(opts Options) (binmeta.BinCandidate, error) {
	res, err := resolveResult(opts)
	if err != nil {
		return binmeta.BinCandidate{}, err
	}
	return res.Candidate, nil
}

func resolveResult(opts Options) (Result, error) {
	var empty Result
	command := strings.TrimSpace(opts.Command)
	if command == "" {
		return empty, apperr.New(apperr.Usage, "binresolve", "", "binary selector required")
	}
	levels := importerLevels(opts.ProjectRoot, opts.PackageDir, opts.ImporterRel)
	for _, lvl := range levels {
		res, err := resolveAtLevel(opts, lvl, command)
		if err != nil {
			return empty, err
		}
		if len(res.Ambiguity) > 0 {
			deps := binmeta.DependencyNames(recordsFromCandidates(res.Ambiguity))
			return res, apperr.New(apperr.Usage, "binresolve", command, AmbiguityMessage(command, deps))
		}
		if res.Candidate.Command != "" {
			return res, nil
		}
	}
	empty.MissMessage = MissMessage(command)
	return empty, apperr.New(apperr.NotFound, "binresolve", command, empty.MissMessage)
}

func resolveAtLevel(opts Options, lvl importerLevel, command string) (Result, error) {
	var res Result
	doc := loadDoc(opts, lvl.NodeModules)
	if doc != nil && doc.LayoutMode == binmeta.LayoutPnP {
		if cand, amb, err := resolvePnP(opts, lvl, command, doc); err != nil {
			return res, err
		} else if len(amb) > 0 {
			res.Ambiguity = amb
			return res, nil
		} else if cand.Command != "" {
			res.Candidate = cand
			return res, nil
		}
	}
	if doc != nil && opts.GenerationID != "" && opts.Fingerprint != "" && binmeta.Stale(doc, opts.GenerationID, opts.Fingerprint) {
		if opts.RequireVerified {
			return res, apperr.New(apperr.Integrity, "binresolve", command, "stale bin metadata generation")
		}
		doc = nil
	}
	if doc != nil && opts.GenerationID != "" && opts.Fingerprint != "" && !binmeta.GenerationMatches(doc, opts.GenerationID, opts.Fingerprint) {
		if opts.RequireVerified {
			return res, apperr.New(apperr.Integrity, "binresolve", command, "bin metadata generation mismatch")
		}
		doc = nil
	}
	recs := filterRecords(doc, command, opts.PackageFilter)
	if len(recs) > 1 {
		res.Ambiguity = candidatesFromRecords(recs, lvl.NodeModules, true)
		return res, nil
	}
	if len(recs) == 1 {
		cand, err := candidateFromRecord(recs[0], lvl.NodeModules, command)
		if err != nil {
			return res, err
		}
		if err := ValidateCandidate(cand, lvl.NodeModules); err != nil {
			return res, err
		}
		if !ShimMatchesMetadata(recs[0], cand.ShimPath) {
			return res, apperr.New(apperr.Integrity, "binresolve", command, "bin metadata does not match filesystem shim")
		}
		res.Candidate = cand
		return res, nil
	}
	if opts.RequireVerified {
		return res, nil
	}
	if !opts.AllowUnowned {
		return res, nil
	}
	shim, err := exactShimPath(lvl.NodeModules, command)
	if err != nil {
		return res, err
	}
	if shim == "" {
		return res, nil
	}
	if doc != nil && len(binmeta.VerifiedOwners(doc, command)) > 0 {
		return res, apperr.New(apperr.Integrity, "binresolve", command, "canonical metadata conflict with compatibility shim")
	}
	cand, err := candidateFromShim(shim, command, lvl.NodeModules)
	if err != nil {
		return res, err
	}
	if err := ValidateCandidate(cand, lvl.NodeModules); err != nil {
		return res, err
	}
	res.Candidate = cand
	res.UsedFallback = true
	return res, nil
}

func loadDoc(opts Options, nodeModules string) *binmeta.Document {
	if opts.RequestCache != nil {
		if doc, ok := opts.RequestCache[nodeModules]; ok {
			return doc
		}
	}
	doc, _ := binmeta.Read(nodeModules)
	if opts.RequestCache != nil {
		opts.RequestCache[nodeModules] = doc
	}
	return doc
}

func filterRecords(doc *binmeta.Document, command, pkg string) []binmeta.Record {
	recs := binmeta.RecordsForCommand(doc, command)
	if pkg == "" {
		return recs
	}
	var out []binmeta.Record
	for _, rec := range recs {
		if rec.DependencyName == pkg {
			out = append(out, rec)
		}
	}
	return out
}

func candidateFromRecord(rec binmeta.Record, nodeModules, command string) (binmeta.BinCandidate, error) {
	shim, err := exactShimPath(nodeModules, command)
	if err != nil {
		return binmeta.BinCandidate{}, err
	}
	if shim == "" {
		if filepath.IsAbs(rec.MaterializedShim) {
			shim = rec.MaterializedShim
		} else {
			shim = filepath.Join(nodeModules, rec.MaterializedShim)
		}
	}
	target := filepath.Join(rec.PackageDir, rec.DeclaredBin)
	if rec.PackageDir != "" {
		target = filepath.Join(rec.PackageDir, filepath.Base(rec.DeclaredBin))
	}
	return binmeta.BinCandidate{
		Command:           command,
		DependencyName:    rec.DependencyName,
		ResolvedPackage:   rec.ResolvedPackage,
		PackageDir:        rec.PackageDir,
		ShimPath:          shim,
		TargetPath:        target,
		OwnershipVerified: rec.OwnershipVerified,
	}, nil
}

func candidatesFromRecords(recs []binmeta.Record, nodeModules string, verified bool) []binmeta.BinCandidate {
	var out []binmeta.BinCandidate
	for _, rec := range recs {
		cand, err := candidateFromRecord(rec, nodeModules, rec.DeclaredBin)
		if err != nil {
			continue
		}
		cand.OwnershipVerified = verified && rec.OwnershipVerified
		out = append(out, cand)
	}
	binmeta.SortCandidates(out)
	return out
}

func recordsFromCandidates(cands []binmeta.BinCandidate) []binmeta.Record {
	recs := make([]binmeta.Record, 0, len(cands))
	for _, c := range cands {
		recs = append(recs, binmeta.Record{
			DependencyName:  c.DependencyName,
			ResolvedPackage: c.ResolvedPackage,
			PackageDir:      c.PackageDir,
			DeclaredBin:     c.Command,
		})
	}
	return recs
}

func exactShimPath(nodeModules, command string) (string, error) {
	binDir := filepath.Join(nodeModules, ".bin")
	if runtime.GOOS == "windows" {
		for _, name := range []string{command + ".cmd", command + ".exe", command} {
			p := filepath.Join(binDir, name)
			if st, err := os.Lstat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
		return "", nil
	}
	p := filepath.Join(binDir, command)
	if st, err := os.Lstat(p); err == nil && !st.IsDir() {
		return p, nil
	}
	return "", nil
}

func candidateFromShim(shim, command, nodeModules string) (binmeta.BinCandidate, error) {
	target := shim
	return binmeta.BinCandidate{
		Command:           command,
		ShimPath:          shim,
		TargetPath:        target,
		OwnershipVerified: false,
		PackageDir:        filepath.Dir(filepath.Dir(nodeModules)),
	}, nil
}

// CheapVerifiedHint reports whether a verified metadata owner exists without full resolution.
func CheapVerifiedHint(opts Options) (command string, found bool, err error) {
	levels := importerLevels(opts.ProjectRoot, opts.PackageDir, opts.ImporterRel)
	for _, lvl := range levels {
		doc := loadDoc(opts, lvl.NodeModules)
		if doc == nil {
			continue
		}
		if opts.GenerationID != "" && opts.Fingerprint != "" && !binmeta.GenerationMatches(doc, opts.GenerationID, opts.Fingerprint) {
			continue
		}
		recs := binmeta.VerifiedOwners(doc, opts.Command)
		if len(recs) == 1 {
			shim, err := exactShimPath(lvl.NodeModules, opts.Command)
			if err != nil || shim == "" {
				continue
			}
			if !ShimMatchesMetadata(recs[0], shim) {
				return "", false, apperr.New(apperr.Integrity, "binresolve", opts.Command, "bin metadata does not match filesystem shim")
			}
			return opts.Command, true, nil
		}
		if len(recs) > 1 {
			return "", false, apperr.New(apperr.Usage, "binresolve", opts.Command,
				AmbiguityMessage(opts.Command, binmeta.DependencyNames(recs)))
		}
	}
	return "", false, nil
}
