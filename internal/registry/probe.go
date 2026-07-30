package registry

import (
	"sync/atomic"

	"github.com/mewisme/mew/internal/fetch"
)

var metadataCallCount atomic.Int64

// TestProbeReset clears all registry probe counters.
func TestProbeReset() {
	metadataCallCount.Store(0)
	fetch.TestProbeArtifactReset()
}

// TestProbeCalls returns total registry-related operations since last reset.
func TestProbeCalls() int64 {
	return TestProbeMetadataCalls() + TestProbeArtifactCalls()
}

// TestProbeMetadataCalls returns metadata endpoint calls since last reset.
func TestProbeMetadataCalls() int64 {
	return metadataCallCount.Load()
}

// TestProbeArtifactCalls returns artifact endpoint calls since last reset.
func TestProbeArtifactCalls() int64 {
	return fetch.TestProbeArtifactCalls()
}

func noteMetadataCall() {
	metadataCallCount.Add(1)
}
