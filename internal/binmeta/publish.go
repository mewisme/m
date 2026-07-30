package binmeta

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/linker"
)

// PublishInput describes one bins metadata publish during install/link.
type PublishInput struct {
	NodeModules      string
	ImporterIdentity string
	GenerationID     string
	LayoutMode       LayoutMode
	Sources          []linker.BinSource
}

// BuildDocument constructs a bins metadata document from link bin sources.
func BuildDocument(in PublishInput) (*Document, error) {
	if in.NodeModules == "" {
		return nil, fmt.Errorf("missing node_modules")
	}
	if in.GenerationID == "" {
		return nil, fmt.Errorf("missing generationID")
	}
	mode := in.LayoutMode
	if mode == "" {
		mode = LayoutHoisted
	}
	records := make([]Record, 0, len(in.Sources))
	for _, src := range in.Sources {
		if src.Cmd == "" {
			continue
		}
		shim := materializedShimPath(in.NodeModules, src.Cmd)
		records = append(records, Record{
			DependencyName:    filepath.Base(src.PackageDir),
			ResolvedPackage:   src.PackageDir,
			PackageDir:        src.PackageDir,
			DeclaredBin:       src.Cmd,
			MaterializedShim:  shim,
			OwnershipVerified: true,
		})
	}
	SortRecords(records)
	fp := fingerprint(records, in.ImporterIdentity, string(mode))
	return &Document{
		SchemaVersion:    SchemaVersion,
		GenerationID:     in.GenerationID,
		ImporterIdentity: in.ImporterIdentity,
		LayoutMode:       mode,
		Fingerprint:      fp,
		Records:          records,
	}, nil
}

func materializedShimPath(_ string, cmd string) string {
	// ponytail: extension chosen by linker platform shims; metadata stores logical command name.
	// Relative to node_modules so metadata survives staging→final cache rename.
	return filepath.Join(".bin", cmd)
}

func fingerprint(records []Record, importer, layout string) string {
	h := sha256.New()
	fmt.Fprintf(h, "importer=%s\nlayout=%s\n", importer, layout)
	for _, rec := range records {
		fmt.Fprintf(h, "%s|%s|%s|%s|%s\n",
			rec.DeclaredBin, rec.DependencyName, rec.ResolvedPackage, rec.PackageDir, rec.MaterializedShim)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Publish writes validated bins metadata for one node_modules tree.
func Publish(in PublishInput) error {
	doc, err := BuildDocument(in)
	if err != nil {
		return err
	}
	if err := Validate(doc); err != nil {
		return err
	}
	return Write(in.NodeModules, doc)
}

// RecordsForCommand returns metadata records for one command at a level.
func RecordsForCommand(doc *Document, command string) []Record {
	if doc == nil {
		return nil
	}
	var out []Record
	for _, rec := range doc.Records {
		if rec.DeclaredBin == command {
			out = append(out, rec)
		}
	}
	SortRecords(out)
	return out
}

// VerifiedOwners returns verified ownership records for a command.
func VerifiedOwners(doc *Document, command string) []Record {
	recs := RecordsForCommand(doc, command)
	var out []Record
	for _, rec := range recs {
		if rec.OwnershipVerified {
			out = append(out, rec)
		}
	}
	return out
}

// DependencyNames returns sorted unique dependency names providing command.
func DependencyNames(recs []Record) []string {
	seen := map[string]struct{}{}
	var names []string
	for _, rec := range recs {
		name := strings.TrimSpace(rec.DependencyName)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
