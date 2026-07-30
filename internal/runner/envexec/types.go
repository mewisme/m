package envexec

import (
	"context"
	"io"

	"github.com/mewisme/mew/internal/runner/dlx"
)

// SourceKind identifies which environment provider owns a request.
type SourceKind string

const (
	SourceProject  SourceKind = "project"
	SourceDLX      SourceKind = "dlx"
	SourceSnapshot SourceKind = "snapshot"
	SourceCapsule  SourceKind = "capsule"
)

// ExecutionRequest is the tagged unified execution input.
// Exactly one concrete source must be supplied via Source.
type ExecutionRequest struct {
	Source  SourceRequest
	Command CommandRequest
	Policy  ExecutionPolicy
	IO      IOStreams

	// Recursive enables workspace-recursive execution for project sources only.
	Recursive bool
	// Filters selects workspace importers for project sources.
	Filters []string

	// Suspend / Resume pause presentation around child launch (optional).
	Suspend func(context.Context) error
	Resume  func(context.Context) error
}

// SourceRequest is a tagged execution source. Implementations validate
// source-specific fields; cross-source rules are enforced in validate.go.
type SourceRequest interface {
	Kind() SourceKind
	validateSource() error
}

// ProjectSource executes against the local project importer graph.
type ProjectSource struct {
	CWD          string
	ImporterHint string
}

func (ProjectSource) Kind() SourceKind { return SourceProject }

func (s ProjectSource) validateSource() error {
	if s.CWD == "" {
		return usageError("project source requires cwd")
	}
	return nil
}

// DLXMode distinguishes mx invocation shapes.
type DLXMode int

const (
	// DLXModeExplicitPackages is mx Mode B: -p pkg... <bin>.
	DLXModeExplicitPackages DLXMode = iota
	// DLXModePackageCommand is mx Mode A: <package-spec> <bin>.
	DLXModePackageCommand
)

// DLXSource executes ephemeral package environments.
type DLXSource struct {
	Packages []dlx.PackageSpec
	Mode     DLXMode
	Offline  bool
	Yes      bool
}

func (DLXSource) Kind() SourceKind { return SourceDLX }

func (s DLXSource) validateSource() error {
	if len(s.Packages) == 0 {
		return usageError("dlx source requires at least one package spec")
	}
	return nil
}

// SnapshotSource executes from a project-local snapshot record.
type SnapshotSource struct {
	ProjectRoot string
	SnapshotID  string
}

func (SnapshotSource) Kind() SourceKind { return SourceSnapshot }

func (s SnapshotSource) validateSource() error {
	if s.SnapshotID == "" {
		return usageError("snapshot source requires snapshot id")
	}
	return nil
}

// CapsuleSource executes from a user-supplied capsule archive.
type CapsuleSource struct {
	Path string
}

func (CapsuleSource) Kind() SourceKind { return SourceCapsule }

func (s CapsuleSource) validateSource() error {
	if s.Path == "" {
		return usageError("capsule source requires path")
	}
	return nil
}

// CommandRequest selects the child binary. Name is required for execution.
// OwnerDependency names the importer-visible dependency owner, not a workspace
// filter. Child args are preserved verbatim and never affect identity.
type CommandRequest struct {
	Name              string
	OwnerDependency   string
	RequireOwnerMatch bool
	Args              []string
}

// NetworkPolicy controls registry and network access during plan/materialize.
type NetworkPolicy string

const (
	NetworkForbidden    NetworkPolicy = "forbidden"
	NetworkMetadataOnly NetworkPolicy = "metadata-only"
	NetworkAllowed      NetworkPolicy = "allowed"
)

// MutationPolicy controls which filesystem roots may be mutated.
type MutationPolicy string

const (
	MutationProjectForbidden MutationPolicy = "project-forbidden"
	MutationCacheOnly        MutationPolicy = "cache-only"
)

// VerificationPolicy controls warm-environment verification strictness.
type VerificationPolicy string

const (
	VerificationCompatible VerificationPolicy = "compatible"
	VerificationRequired   VerificationPolicy = "required"
)

// ExecutionPolicy is the typed policy bundle for a request.
type ExecutionPolicy struct {
	Network      NetworkPolicy
	Mutation     MutationPolicy
	Verification VerificationPolicy
}

// IOStreams carries stdio handles for providers that need consent or diagnostics I/O.
type IOStreams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}
