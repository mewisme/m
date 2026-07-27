package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/linker/planner"
)

// FilesystemProbeReport is the result of m development doctor filesystem.
type FilesystemProbeReport struct {
	StoreRoot string
	DestRoot  string
	Caps      planner.Capabilities
}

// DoctorFilesystem probes link capabilities between store and node_modules.
func DoctorFilesystem(ctx context.Context, ac *Context) (FilesystemProbeReport, error) {
	var rep FilesystemProbeReport
	if err := ctx.Err(); err != nil {
		return rep, err
	}
	if ac == nil || ac.Config == nil {
		return rep, apperr.New(apperr.Internal, "app.doctor.filesystem", "", "missing app context")
	}
	storeRoot, err := config.StoreRoot(ac.Config)
	if err != nil {
		return rep, err
	}
	rep.StoreRoot = storeRoot
	rep.DestRoot = "node_modules"
	if p, err := OpenProject(ctx, ac); err == nil {
		rep.DestRoot = filepath.Join(p.Root, "node_modules")
	}
	rep.Caps, err = planner.ProbeCached(config.CacheRoot(ac.Config), rep.StoreRoot, rep.DestRoot)
	return rep, err
}

// FormatFilesystemProbe returns human-readable probe output.
func FormatFilesystemProbe(rep FilesystemProbeReport) string {
	return fmt.Sprintf("src=%s\ndest=%s\nsameVolume=%v hardlink=%v reflink=%v symlink=%v junction=%v\n",
		rep.StoreRoot, rep.DestRoot,
		rep.Caps.SameVolume, rep.Caps.Hardlink, rep.Caps.Reflink, rep.Caps.Symlink, rep.Caps.Junction)
}
