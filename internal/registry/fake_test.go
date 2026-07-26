package registry_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/registry"
)

type fakeRegistry struct{}

func (fakeRegistry) Metadata(context.Context, string, string) (*registry.PackageMetadata, error) {
	return &registry.PackageMetadata{}, nil
}

var _ registry.Registry = fakeRegistry{}

func TestFakeRegistrySatisfiesInterface(t *testing.T) {
	var r registry.Registry = fakeRegistry{}
	if _, err := r.Metadata(context.Background(), "left-pad", "1.0.0"); err != nil {
		t.Fatal(err)
	}
}
