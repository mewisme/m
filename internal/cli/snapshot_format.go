package cli

import (
	"fmt"
	"io"

	"github.com/mewisme/mew/internal/snapshot"
)

type snapshotEntry struct {
	Snapshot snapshot.Snapshot `json:",inline"`
	Delta    string            `json:"delta,omitempty"`
}

func formatSnapshotLine(w io.Writer, s snapshot.Snapshot, prev *snapshot.Snapshot) error {
	delta := snapshotDeltaSummary(s, prev)
	if delta == "" {
		_, err := fmt.Fprintf(w, "%s  %s  %s\n",
			s.ID, s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), shortDigest(s.GraphDigest))
		return err
	}
	_, err := fmt.Fprintf(w, "%s  %s  %s  %s\n",
		s.ID, s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), shortDigest(s.GraphDigest), delta)
	return err
}

func snapshotDeltaSummary(current snapshot.Snapshot, older *snapshot.Snapshot) string {
	if older == nil {
		return "initial"
	}
	if current.GraphDigest == older.GraphDigest {
		return "unchanged graph"
	}
	return "graph changed"
}

func snapshotEntriesWithDelta(list []snapshot.Snapshot) []snapshotEntry {
	out := make([]snapshotEntry, len(list))
	for i, s := range list {
		var older *snapshot.Snapshot
		if i+1 < len(list) {
			older = &list[i+1]
		}
		out[i] = snapshotEntry{
			Snapshot: s,
			Delta:    snapshotDeltaSummary(s, older),
		}
	}
	return out
}
