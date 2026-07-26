// Package registry provides registry clients, authentication, and metadata access.
package registry

import "context"

// Registry fetches package metadata from a configured registry.
type Registry interface {
	Metadata(ctx context.Context, name, version string) (*PackageMetadata, error)
}
