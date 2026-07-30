package envexec

import (
	"github.com/mewisme/mew/internal/apperr"
)

var (
	// ErrUnimplemented reports that a provider phase is not yet implemented.
	ErrUnimplemented = apperr.New(apperr.Unimplemented, "envexec", "", "not implemented")
)

func usageError(msg string) error {
	return apperr.New(apperr.Usage, "envexec.validate", "", msg)
}

// ValidateRequest enforces the source combination matrix and command requirements.
func ValidateRequest(req ExecutionRequest) error {
	if req.Source == nil {
		return usageError("missing execution source")
	}
	if err := req.Source.validateSource(); err != nil {
		return err
	}
	if req.Command.Name == "" {
		if src, ok := req.Source.(DLXSource); !ok || src.Mode != DLXModePackageCommand {
			return usageError("missing command selector")
		}
	}

	kind := req.Source.Kind()

	if req.Recursive && kind != SourceProject {
		return usageError("recursive execution is only supported for project sources")
	}

	switch kind {
	case SourceProject:
		return validateProjectRequest(req)
	case SourceDLX:
		return validateDLXRequest(req)
	case SourceSnapshot:
		return validateSnapshotRequest(req)
	case SourceCapsule:
		return validateCapsuleRequest(req)
	default:
		return usageError("unsupported execution source")
	}
}

func validateProjectRequest(req ExecutionRequest) error {
	if _, ok := req.Source.(ProjectSource); !ok {
		return usageError("invalid project source")
	}
	return nil
}

func validateDLXRequest(req ExecutionRequest) error {
	src, ok := req.Source.(DLXSource)
	if !ok {
		return usageError("invalid dlx source")
	}
	if req.Command.Name == "" && src.Mode != DLXModePackageCommand {
		return usageError("missing command selector")
	}
	if req.Command.RequireOwnerMatch && req.Command.OwnerDependency != "" {
		return usageError("dlx execution does not support project dependency owner selection")
	}
	return nil
}

func validateSnapshotRequest(req ExecutionRequest) error {
	src, ok := req.Source.(SnapshotSource)
	if !ok {
		return usageError("invalid snapshot source")
	}
	if src.ProjectRoot == "" {
		return usageError("snapshot source requires project root")
	}
	if req.Command.OwnerDependency != "" || req.Command.RequireOwnerMatch {
		return usageError("snapshot execution does not support project dependency owner selection")
	}
	return nil
}

func validateCapsuleRequest(req ExecutionRequest) error {
	if _, ok := req.Source.(CapsuleSource); !ok {
		return usageError("invalid capsule source")
	}
	if req.Command.OwnerDependency != "" || req.Command.RequireOwnerMatch {
		return usageError("capsule execution does not support project dependency owner selection")
	}
	return nil
}

// ValidateInspectRequest validates inspect input without requiring a command selector.
func ValidateInspectRequest(req ExecutionRequest) error {
	if req.Source == nil {
		return usageError("missing execution source")
	}
	if err := req.Source.validateSource(); err != nil {
		return err
	}
	kind := req.Source.Kind()
	if req.Recursive && kind != SourceProject {
		return usageError("recursive execution is only supported for project sources")
	}
	switch kind {
	case SourceProject:
		return validateProjectRequest(req)
	case SourceDLX:
		return validateDLXRequest(req)
	case SourceSnapshot:
		return validateSnapshotRequest(req)
	case SourceCapsule:
		return validateCapsuleRequest(req)
	default:
		return usageError("unsupported execution source")
	}
}

// ValidatePolicy rejects unknown policy values.
func ValidatePolicy(policy ExecutionPolicy) error {
	if !isNetworkPolicy(policy.Network) {
		return usageError("invalid network policy")
	}
	if !isMutationPolicy(policy.Mutation) {
		return usageError("invalid mutation policy")
	}
	if !isVerificationPolicy(policy.Verification) {
		return usageError("invalid verification policy")
	}
	return nil
}

func isNetworkPolicy(p NetworkPolicy) bool {
	switch p {
	case NetworkForbidden, NetworkMetadataOnly, NetworkAllowed:
		return true
	default:
		return false
	}
}

func isMutationPolicy(p MutationPolicy) bool {
	switch p {
	case MutationProjectForbidden, MutationCacheOnly:
		return true
	default:
		return false
	}
}

func isVerificationPolicy(p VerificationPolicy) bool {
	switch p {
	case VerificationCompatible, VerificationRequired:
		return true
	default:
		return false
	}
}

// IsUsage reports whether err is a usage validation failure.
func IsUsage(err error) bool {
	return apperr.IsUsage(err)
}
