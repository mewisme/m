package envexec

// LifecycleMaterializationPolicy controls lifecycle script execution during materialization.
type LifecycleMaterializationPolicy string

const (
	// LifecycleForbidden forbids lifecycle scripts during frozen materialization.
	LifecycleForbidden LifecycleMaterializationPolicy = "forbidden"
)

// LockedProviderPolicy returns the default execution policy for a source kind.
func LockedProviderPolicy(kind SourceKind) ExecutionPolicy {
	switch kind {
	case SourceProject:
		return ExecutionPolicy{
			Network:      NetworkForbidden,
			Mutation:     MutationProjectForbidden,
			Verification: VerificationCompatible,
		}
	case SourceDLX:
		// Remote DLX may upgrade network after consent in later phases.
		return ExecutionPolicy{
			Network:      NetworkMetadataOnly,
			Mutation:     MutationCacheOnly,
			Verification: VerificationRequired,
		}
	case SourceSnapshot, SourceCapsule:
		return ExecutionPolicy{
			Network:      NetworkForbidden,
			Mutation:     MutationCacheOnly,
			Verification: VerificationRequired,
		}
	default:
		return ExecutionPolicy{}
	}
}

// LockedLifecyclePolicy returns the lifecycle materialization policy for a source kind.
func LockedLifecyclePolicy(kind SourceKind) LifecycleMaterializationPolicy {
	switch kind {
	case SourceSnapshot, SourceCapsule:
		return LifecycleForbidden
	default:
		return ""
	}
}

// LockedLeasePolicy returns the execution lease policy for a source kind.
func LockedLeasePolicy(kind SourceKind) LeasePolicy {
	switch kind {
	case SourceDLX, SourceSnapshot, SourceCapsule:
		return LeaseSharedCache
	default:
		return LeaseNone
	}
}
