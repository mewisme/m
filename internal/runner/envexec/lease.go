package envexec

import "context"

// LeasePolicy controls shared-cache execution leases.
type LeasePolicy string

const (
	// LeaseNone means no execution lease is acquired.
	LeaseNone LeasePolicy = "none"
	// LeaseSharedCache protects a shared materialized environment from prune.
	LeaseSharedCache LeasePolicy = "shared-cache"
)

// LeaseManager acquires and releases execution leases for shared cache entries.
type LeaseManager interface {
	Acquire(ctx context.Context, identity EnvironmentIdentity, holder string, pid int, token int64) (release func(), err error)
}
