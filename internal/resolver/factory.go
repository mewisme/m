package resolver

import (
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
)

// NewFromApp builds an Engine from effective config and a loaded project.
func NewFromApp(eff *config.Effective, proj *project.Project) (*Engine, error) {
	root := ""
	identity := project.IdentityMew
	if proj != nil {
		root = proj.Root
		identity = proj.Identity
	}
	client, err := registry.NewFromApp(eff, root, identity)
	if err != nil {
		return nil, err
	}
	return NewEngine(client, eff, identity), nil
}
