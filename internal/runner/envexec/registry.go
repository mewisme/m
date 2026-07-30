package envexec

import "fmt"

// ProviderRegistry maps source kinds to environment providers.
type ProviderRegistry struct {
	providers map[SourceKind]EnvironmentProvider
}

// NewProviderRegistry builds a registry from the given providers.
func NewProviderRegistry(providers ...EnvironmentProvider) ProviderRegistry {
	m := make(map[SourceKind]EnvironmentProvider, len(providers))
	for _, p := range providers {
		if p == nil {
			continue
		}
		m[p.Kind()] = p
	}
	return ProviderRegistry{providers: m}
}

// DefaultProviderRegistry returns providers for every source kind.
func DefaultProviderRegistry() ProviderRegistry {
	return NewProviderRegistry(
		ProjectProvider{},
		DLXProvider{},
		SnapshotProvider{},
		CapsuleProvider{},
	)
}

// Provider returns the provider for kind, or nil.
func (r ProviderRegistry) Provider(kind SourceKind) EnvironmentProvider {
	if r.providers == nil {
		return nil
	}
	return r.providers[kind]
}

// Register adds or replaces a provider for its kind.
func (r *ProviderRegistry) Register(p EnvironmentProvider) {
	if r.providers == nil {
		r.providers = make(map[SourceKind]EnvironmentProvider)
	}
	r.providers[p.Kind()] = p
}

func (r ProviderRegistry) providerFor(req ExecutionRequest) (EnvironmentProvider, error) {
	if req.Source == nil {
		return nil, usageError("missing execution source")
	}
	p := r.Provider(req.Source.Kind())
	if p == nil {
		return nil, usageError(fmt.Sprintf("no provider for source %q", req.Source.Kind()))
	}
	return p, nil
}
