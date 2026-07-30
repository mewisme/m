package fetch

import "sync/atomic"

var artifactCallCount atomic.Int64

// TestProbeArtifactReset clears artifact fetch probe counter.
func TestProbeArtifactReset() {
	artifactCallCount.Store(0)
}

// TestProbeArtifactCalls returns artifact endpoint calls since last reset.
func TestProbeArtifactCalls() int64 {
	return artifactCallCount.Load()
}

// NoteArtifactCall records one artifact fetch for tests.
func NoteArtifactCall() {
	artifactCallCount.Add(1)
}
