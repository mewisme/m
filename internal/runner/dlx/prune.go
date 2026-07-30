package dlx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// PruneCandidate is one environment eligible for pruning.
type PruneCandidate struct {
	Digest   string
	Path     string
	LastUsed time.Time
}

// PruneOptions configures cache pruning.
type PruneOptions struct {
	MXCacheDir    string
	RetentionDays int
	OlderThan     time.Duration
	DryRun        bool
}

// PruneEnvironments removes stale environments respecting leases and locks.
func PruneEnvironments(opts PruneOptions) ([]PruneCandidate, error) {
	if opts.MXCacheDir == "" {
		return nil, apperr.New(apperr.Config, "dlx.prune", "", "missing mx cache dir")
	}
	cutoff := time.Now().Add(-time.Duration(opts.RetentionDays) * 24 * time.Hour)
	if opts.OlderThan > 0 {
		cutoff = time.Now().Add(-opts.OlderThan)
	}
	execRoot := filepath.Join(opts.MXCacheDir, "exec")
	ents, err := os.ReadDir(execRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "dlx.prune", execRoot, err)
	}
	var removed []PruneCandidate
	for _, ent := range ents {
		if !ent.IsDir() || strings.Contains(ent.Name(), ".staging.") {
			continue
		}
		digest := ent.Name()
		if HasLiveLeases(opts.MXCacheDir, digest) {
			continue
		}
		envDir := filepath.Join(execRoot, digest)
		last := lastUsedAt(opts.MXCacheDir, envDir, digest)
		if !last.Before(cutoff) {
			continue
		}
		cand := PruneCandidate{Digest: digest, Path: envDir, LastUsed: last}
		if !opts.DryRun {
			_ = os.RemoveAll(envDir)
			_ = os.Remove(AccessPath(opts.MXCacheDir, digest))
		}
		removed = append(removed, cand)
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i].Digest < removed[j].Digest })
	return removed, nil
}

func lastUsedAt(mxCacheDir, envDir, digest string) time.Time {
	if b, err := os.ReadFile(AccessPath(mxCacheDir, digest)); err == nil {
		var rec AccessRecord
		if json.Unmarshal(b, &rec) == nil && rec.LastUsedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, rec.LastUsedAt); err == nil {
				return t
			}
		}
	}
	if b, err := os.ReadFile(ReadyPath(envDir)); err == nil {
		var ready ReadyMarker
		if json.Unmarshal(b, &ready) == nil && ready.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, ready.CreatedAt); err == nil {
				return t
			}
		}
	}
	if st, err := os.Stat(envDir); err == nil {
		return st.ModTime()
	}
	return time.Time{}
}
