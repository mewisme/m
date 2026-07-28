package resolver

import (
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/registry"
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
