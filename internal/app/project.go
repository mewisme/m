package app

import (
	"context"

	"github.com/mewisme/mew/internal/project"
)

// OpenProject discovers and loads the project for the application CWD.
func OpenProject(ctx context.Context, ac *Context) (*project.Project, error) {
	if ac == nil {
		cwd := ""
		return project.Open(ctx, cwd)
	}
	if ctx == nil {
		ctx = ac.Ctx
	}
	return project.Open(ctx, ac.CWD)
}
