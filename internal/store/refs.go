package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/transaction"
)

// CollectReferencedIntegrities scans roots for store-manifest.json files and active
// transaction journals that reference package integrities.
func CollectReferencedIntegrities(roots []string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return err
			}
			base := filepath.Base(path)
			switch base {
			case "store-manifest.json":
				if isTxnStagedManifest(path) {
					return nil
				}
				addManifestIntegrities(path, out)
			case transaction.JournalName, transaction.JournalNameV1:
				addTxnJournalIntegrities(path, out)
			}
			return nil
		})
	}
	return out, nil
}

func isTxnStagedManifest(path string) bool {
	slash := filepath.ToSlash(path)
	return strings.Contains(slash, "/.mew/txn/") && strings.Contains(slash, "/stage/.mew/store-manifest.json")
}

func addManifestIntegrities(path string, out map[string]struct{}) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var doc struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return
	}
	for _, k := range doc.Packages {
		if k != "" {
			out[k] = struct{}{}
		}
	}
}

func addTxnJournalIntegrities(journalPath string, out map[string]struct{}) {
	raw, err := os.ReadFile(journalPath)
	if err != nil {
		return
	}
	doc, err := transaction.Decode(raw)
	if err != nil {
		return
	}
	if doc.State == transaction.StateCommitted || doc.State == transaction.StateAborted {
		return
	}
	txnDir := filepath.Dir(journalPath)
	stageManifest := filepath.Join(txnDir, "stage", ".mew", "store-manifest.json")
	addManifestIntegrities(stageManifest, out)
}

// PruneCandidates returns package keys sorted for deterministic prune dry-run.
func PruneCandidates(s *PackageStore, refs map[string]struct{}) ([]PackageKey, error) {
	keys, err := s.ListPackageKeys()
	if err != nil {
		return nil, err
	}
	var candidates []PackageKey
	for _, key := range keys {
		if _, ok := refs[key.Integrity()]; ok {
			continue
		}
		pkgDir := s.PackagePath(key)
		if HasImportLock(pkgDir) {
			continue
		}
		candidates = append(candidates, key)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].String() < candidates[j].String()
	})
	return candidates, nil
}
