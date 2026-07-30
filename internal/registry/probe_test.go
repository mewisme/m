package registry

import (
	"testing"

	"github.com/mewisme/mew/internal/fetch"
)

func TestProbeMetadataVsArtifact(t *testing.T) {
	TestProbeReset()
	noteMetadataCall()
	if TestProbeMetadataCalls() != 1 || TestProbeArtifactCalls() != 0 {
		t.Fatalf("meta=%d artifact=%d", TestProbeMetadataCalls(), TestProbeArtifactCalls())
	}
	fetch.NoteArtifactCall()
	if TestProbeArtifactCalls() != 1 {
		t.Fatalf("artifact=%d", TestProbeArtifactCalls())
	}
}
