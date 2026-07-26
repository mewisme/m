package resolver

import (
	"os"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
)

// NewFromApp builds an Engine from effective config and a loaded project.
func NewFromApp(eff *config.Effective, proj *project.Project, environ []string) (*Engine, error) {
	if environ == nil {
		environ = os.Environ()
	}
	root := ""
	identity := project.IdentityMew
	if proj != nil {
		root = proj.Root
		identity = proj.Identity
	}
	client, err := registry.NewFromApp(eff, root, identity, environ)
	if err != nil {
		return nil, err
	}
	return NewEngine(client, eff, identity), nil
}
