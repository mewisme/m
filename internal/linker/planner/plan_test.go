package planner_test

import (
	"testing"

	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/linker/planner"
)

func TestPlanFileCrossDeviceUsesCopy(t *testing.T) {
	caps := planner.Capabilities{SameVolume: false, Hardlink: true, Reflink: true}
	op := planner.PlanFile("/src/a", "/dest/a", caps)
	if op.Kind != linker.OpCopy {
		t.Fatalf("kind=%s want copy", op.Kind)
	}
}

func TestPlanPackageLinkPrefersReflink(t *testing.T) {
	caps := planner.Capabilities{SameVolume: true, Hardlink: true, Reflink: true}
	op := planner.PlanPackageLink("/src/pkg", "/dest/pkg", caps)
	if op.Kind != linker.OpReflink {
		t.Fatalf("kind=%s want reflink", op.Kind)
	}
}

func TestPlanPackageLinkCopyWhenNoReflink(t *testing.T) {
	caps := planner.Capabilities{SameVolume: true, Hardlink: true, Reflink: false}
	op := planner.PlanPackageLink("/src/pkg", "/dest/pkg", caps)
	if op.Kind != linker.OpCopy {
		t.Fatalf("kind=%s want copy", op.Kind)
	}
}
