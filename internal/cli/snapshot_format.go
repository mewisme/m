package cli

import (
	"github.com/mewisme/mew/internal/snapshot"
)

type snapshotEntry struct {
	Snapshot snapshot.Snapshot `json:",inline"`
	Delta    string            `json:"delta,omitempty"`
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
