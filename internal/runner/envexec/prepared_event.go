package envexec

import (
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

const preparedEventSchemaVersion = 1

// PreparedEventCacheState is the non-overlapping cacheState enum for environment-prepared v1.
type PreparedEventCacheState string

const (
	PreparedCacheProject   PreparedEventCacheState = "project"
	PreparedCacheWarmHit   PreparedEventCacheState = "warm-hit"
	PreparedCacheColdBuilt PreparedEventCacheState = "cold-built"
	PreparedCacheEphemeral PreparedEventCacheState = "ephemeral"
)

func preparedEventCacheState(env PreparedEnvironment) PreparedEventCacheState {
	switch env.CacheState {
	case CacheProject:
		return PreparedCacheProject
	case CacheWarm:
		return PreparedCacheWarmHit
	case CacheEphemeral:
		return PreparedCacheEphemeral
	default:
		if env.SharedCache {
			return PreparedCacheWarmHit
		}
		return PreparedCacheColdBuilt
	}
}

func buildEnvironmentPreparedEvent(req ExecutionRequest, env PreparedEnvironment, prepareDuration time.Duration) (diagnostics.EnvironmentPreparedEvent, error) {
	identity := BareDigestHex(env.Identity.IdentityDigest())
	graph := BareDigestHex(env.Identity.GraphDigest)
	if graph == "" {
		graph = BareDigestHex(env.Identity.MaterialDigest)
	}
	if !digestHex64.MatchString(identity) || !digestHex64.MatchString(graph) {
		return diagnostics.EnvironmentPreparedEvent{}, apperr.New(apperr.Integrity, "envexec.environment-prepared", "", "missing identity or graph digest")
	}
	return diagnostics.EnvironmentPreparedEvent{
		V:                 preparedEventSchemaVersion,
		Type:              "environment-prepared",
		Source:            string(env.Source),
		IdentityDigest:    identity,
		GraphDigest:       graph,
		CacheState:        string(preparedEventCacheState(env)),
		NetworkUsed:       networkUsedForRequest(req),
		PrepareDurationMs: prepareDuration.Milliseconds(),
	}, nil
}

func networkUsedForRequest(req ExecutionRequest) bool {
	switch req.Policy.Network {
	case NetworkAllowed, NetworkMetadataOnly:
		return true
	default:
		return false
	}
}

// BuildEnvironmentPreparedEvent exports v1 event construction for conformance certification.
func BuildEnvironmentPreparedEvent(req ExecutionRequest, env PreparedEnvironment) (diagnostics.EnvironmentPreparedEvent, error) {
	return buildEnvironmentPreparedEvent(req, env, 0)
}
