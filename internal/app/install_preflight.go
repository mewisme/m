package app

import (
	"context"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/project"
)

// installPreflight is the set of validations that require no mutation. It runs
// before BeginMutationSession so a rejected install never creates `.mew`, never
// writes a journal, never takes the project mutation lock, and never touches
// the network.
//
// Every check here must be read-only. Anything that creates a directory,
// deletes node_modules, or fetches over the network belongs after the session
// begins.
func installPreflight(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions) error {
	if err := ctx.Err(); err != nil {
		return apperr.Wrap(apperr.Cancelled, "app.install.preflight", "", err)
	}
	if ac == nil || ac.Config == nil {
		return apperr.New(apperr.Internal, "app.install.preflight", "", "missing app context")
	}
	if proj == nil {
		return apperr.New(apperr.Internal, "app.install.preflight", "", "nil project")
	}

	// Option and gating validation first: cheapest, and independent of disk.
	if err := requireWorkspacesGate(ac, opts); err != nil {
		return err
	}

	// Foreign lock validation and Yarn PnP rejection: all read-only.
	if err := validatePnpmLockBeforeTxn(proj); err != nil {
		return err
	}
	if err := validateNpmLockBeforeTxn(proj); err != nil {
		return err
	}
	if err := validateBunLockBeforeTxn(proj); err != nil {
		return err
	}
	if err := rejectBunLockbIfPresent(proj); err != nil {
		return err
	}
	if err := validateYarnLockBeforeTxn(proj); err != nil {
		return err
	}
	if err := gateYarnPnPInstall(proj); err != nil {
		return err
	}

	// Frozen-lock validation reads the incumbent lock and compares it against
	// the manifest; it writes nothing.
	if opts.Frozen && !usesStagedSnapshotInputs(opts) {
		if err := validateFrozenLockForProject(ctx, ac, proj); err != nil {
			return err
		}
	}
	return nil
}
