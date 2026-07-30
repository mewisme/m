package dlx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

const requestIndexSchemaVersion = 1

// RequestIndex maps a request identity to a resolved environment digest.
type RequestIndex struct {
	SchemaVersion              int      `json:"schemaVersion"`
	RequestDigest              string   `json:"requestDigest"`
	ResolvedEnvironmentDigest  string   `json:"resolvedEnvironmentDigest"`
	NormalizedRequestedSpecs   []string `json:"normalizedRequestedSpecs"`
	ResolvedDirectPackageKeys  []string `json:"resolvedDirectPackageIdentities"`
	ResolvedAt                 string   `json:"resolvedAt"`
	SanitizedRegistryOrigin    string   `json:"sanitizedRegistryOrigin"`
	TargetPlatformFingerprint  string   `json:"targetPlatformFingerprint"`
	NodeFingerprint            string   `json:"nodeFingerprint"`
	ResolverPolicyFingerprint  string   `json:"resolverPolicyFingerprint"`
	LifecyclePolicyFingerprint string   `json:"lifecyclePolicyFingerprint"`
	LinkerMode                 string   `json:"linkerMode"`
	TransactionID              string   `json:"transactionId"`
}

// RequestIndexPath returns the on-disk path for a request index entry.
func RequestIndexPath(cacheRoot, requestDigest string) string {
	return filepath.Join(cacheRoot, "requests", "v1", requestDigest+".json")
}

// PublishRequestIndex atomically writes a request-to-resolved mapping.
func PublishRequestIndex(path string, doc RequestIndex) error {
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = requestIndexSchemaVersion
	}
	if doc.ResolvedAt == "" {
		doc.ResolvedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "dlx.request_index", path, err)
	}
	b = append(b, '\n')
	if err := fsx.WriteAtomic(path, b, 0o600); err != nil {
		return apperr.Wrap(apperr.IO, "dlx.request_index", path, err)
	}
	return nil
}

// LoadRequestIndex reads and validates a request index entry.
func LoadRequestIndex(path string) (RequestIndex, error) {
	var empty RequestIndex
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return empty, apperr.Wrap(apperr.NotFound, "dlx.request_index", path, err)
		}
		return empty, apperr.Wrap(apperr.IO, "dlx.request_index", path, err)
	}
	var doc RequestIndex
	if err := json.Unmarshal(b, &doc); err != nil {
		return empty, apperr.Wrap(apperr.Integrity, "dlx.request_index", path, err)
	}
	if doc.SchemaVersion != requestIndexSchemaVersion || doc.RequestDigest == "" || doc.ResolvedEnvironmentDigest == "" {
		return empty, apperr.New(apperr.Integrity, "dlx.request_index", path, "malformed request index")
	}
	return doc, nil
}
