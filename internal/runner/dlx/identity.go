package dlx

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

const consentSchemaVersion = 1

// RequestIdentity serializes metadata resolution for the same mutable request.
type RequestIdentity struct {
	NormalizedSpecs            []string `json:"normalizedSpecs"`
	SanitizedRegistryOrigin    string   `json:"sanitizedRegistryOrigin"`
	ResolverPolicyFingerprint  string   `json:"resolverPolicyFingerprint"`
	TargetPlatformFingerprint  string   `json:"targetPlatformFingerprint"`
	NodeFingerprint            string   `json:"nodeFingerprint,omitempty"`
	LinkerMode                 string   `json:"linkerMode,omitempty"`
	LifecyclePolicyFingerprint string   `json:"lifecyclePolicyFingerprint,omitempty"`
}

// Digest returns a stable hex digest for the request identity.
func (r RequestIdentity) Digest() string {
	return digestCanonical(r)
}

// ResolvedEnvironmentIdentity is the immutable cache environment identity.
// Command and child args are intentionally excluded.
type ResolvedEnvironmentIdentity struct {
	GraphDigest                string            `json:"graphDigest"`
	Packages                   []ResolvedPackage `json:"packages"`
	SanitizedRegistryOrigin    string            `json:"sanitizedRegistryOrigin"`
	TargetPlatformFingerprint  string            `json:"targetPlatformFingerprint"`
	NodeFingerprint            string            `json:"nodeFingerprint"`
	LinkerMode                 string            `json:"linkerMode"`
	LifecyclePolicyFingerprint string            `json:"lifecyclePolicyFingerprint"`
	ResolverPolicyFingerprint  string            `json:"resolverPolicyFingerprint"`
	OptionalDepsFingerprint    string            `json:"optionalDepsFingerprint,omitempty"`
}

// ResolvedPackage is one resolved package in the environment graph.
type ResolvedPackage struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Integrity string `json:"integrity,omitempty"`
	Key       string `json:"key"`
}

// Digest returns a stable hex digest for the resolved environment.
func (r ResolvedEnvironmentIdentity) Digest() string {
	return digestCanonical(r)
}

// ConsentKey binds fetch/execute consent to an immutable environment and command.
type ConsentKey struct {
	SchemaVersion              int    `json:"schemaVersion"`
	EnvironmentDigest          string `json:"environmentDigest"`
	Command                    string `json:"command"`
	Owner                      string `json:"owner,omitempty"`
	SanitizedRegistryOrigin    string `json:"sanitizedRegistryOrigin"`
	LifecyclePolicyFingerprint string `json:"lifecyclePolicyFingerprint"`
	TargetPlatformFingerprint  string `json:"targetPlatformFingerprint"`
}

// Digest returns a stable hex digest for consent lookup.
func (c ConsentKey) Digest() string {
	return digestCanonical(c)
}

// NewConsentKey builds a consent key for the given environment and invocation.
func NewConsentKey(env ResolvedEnvironmentIdentity, command, owner string) ConsentKey {
	return ConsentKey{
		SchemaVersion:              consentSchemaVersion,
		EnvironmentDigest:          env.Digest(),
		Command:                    command,
		Owner:                      owner,
		SanitizedRegistryOrigin:    env.SanitizedRegistryOrigin,
		LifecyclePolicyFingerprint: env.LifecyclePolicyFingerprint,
		TargetPlatformFingerprint:  env.TargetPlatformFingerprint,
	}
}

func digestCanonical(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		b = []byte{}
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func SortSpecs(specs []PackageSpec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Raw
	}
	sort.Strings(out)
	return out
}
