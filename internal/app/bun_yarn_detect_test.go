package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/bun"
	"github.com/mewisme/mew/internal/compat/yarn"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/testkit"
)

func TestDetectBunLockFixture(t *testing.T) {
	root := testkit.ModuleRoot(t)
	lockPath := filepath.Join(root, "fixtures", "locks", "bun", "v1-basic", "bun.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	det, err := detectBunLock(data)
	if err != nil {
		t.Fatal(err)
	}
	if det.Format != bun.FormatV1 {
		t.Fatalf("format=%s", det.Format)
	}
}

func TestDetectYarnLockVariants(t *testing.T) {
	root := testkit.ModuleRoot(t)
	classic, _ := os.ReadFile(filepath.Join(root, "fixtures", "locks", "yarn", "classic-v1", "yarn.lock"))
	det, err := detectYarnLock(classic, "")
	if err != nil {
		t.Fatal(err)
	}
	if det.ProducerMajor != 1 {
		t.Fatalf("major=%d", det.ProducerMajor)
	}
	nmDir := filepath.Join(root, "fixtures", "locks", "yarn", "berry-nm")
	nm, _ := os.ReadFile(filepath.Join(nmDir, "yarn.lock"))
	det, err = detectYarnLock(nm, nmDir)
	if err != nil {
		t.Fatal(err)
	}
	if det.Format == "" {
		t.Fatal("expected berry format")
	}
}

func TestGateYarnPnPInstallFixture(t *testing.T) {
	root := testkit.ModuleRoot(t)
	proj := &project.Project{
		Root:     filepath.Join(root, "fixtures", "locks", "yarn", "berry-pnp-readonly"),
		Identity: project.IdentityYarn,
	}
	err := gateYarnPnPInstall(proj)
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.PNPUnsupported {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestYarnBerryNMInstallNotBlocked(t *testing.T) {
	root := testkit.ModuleRoot(t)
	proj := &project.Project{
		Root:     filepath.Join(root, "fixtures", "locks", "yarn", "berry-nm"),
		Identity: project.IdentityYarn,
	}
	if err := gateYarnPnPInstall(proj); err != nil {
		t.Fatal(err)
	}
	_ = yarn.VariantBerryNM
}
