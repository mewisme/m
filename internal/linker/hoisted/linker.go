package hoisted

import (
	"context"

	"github.com/mewisme/mew/internal/linker"
)

// Apply is a package-level helper for callers that already have a plan.
func Apply(ctx context.Context, plan *linker.Plan) error {
	return linker.Apply(ctx, plan)
}
