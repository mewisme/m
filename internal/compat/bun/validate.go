package bun

import (
	"fmt"

	"github.com/mewisme/mew/internal/lockfile"
)

// ValidateSupported rejects unsupported bun.lock versions.
func ValidateSupported(doc *Document) error {
	if doc == nil {
		return lockfile.NewUnsupported("bun.validate", "bun.lock", "nil document")
	}
	switch doc.LockfileVersion {
	case 0, 1:
		return nil
	default:
		return lockfile.NewUnsupported("bun.validate", "bun.lock",
			fmt.Sprintf("unsupported lockfileVersion %d", doc.LockfileVersion))
	}
}

// DetectFromDocument returns detection metadata for a bun lock document.
func DetectFromDocument(doc *Document) lockfile.Detection {
	major := doc.LockfileVersion
	if major == 0 {
		major = 1
	}
	return lockfile.Detection{
		Format:        FormatV1,
		ProducerMajor: major,
		Confidence:    lockfile.DetectionCertain,
		Evidence:      []string{"lockfileVersion"},
	}
}
