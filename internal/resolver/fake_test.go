package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/resolver"
)

type fakeResolver struct{}

func (fakeResolver) Resolve(context.Context, string, resolver.ResolveOptions) (*resolver.Resolution, error) {
	return &resolver.Resolution{}, nil
}

var _ resolver.Resolver = fakeResolver{}

func TestFakeResolverSatisfiesInterface(t *testing.T) {
	var r resolver.Resolver = fakeResolver{}
	if _, err := r.Resolve(context.Background(), ".", resolver.ResolveOptions{}); err != nil {
		t.Fatal(err)
	}
}
