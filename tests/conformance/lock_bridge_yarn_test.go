package conformance_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/yarn"
	_ "github.com/mewisme/mew/internal/compat/yarn"
	"github.com/mewisme/mew/internal/compat/yarn/berry"
	"github.com/mewisme/mew/internal/compat/yarn/classic"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func TestLockBridgeYarnClassicFixture(t *testing.T) {
	root := moduleRoot(t)
	lockPath := filepath.Join(root, "fixtures", "locks", "yarn", "classic-v1", "yarn.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := classic.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Detection.Format != classic.FormatClassic {
		t.Fatalf("format=%s", doc.Detection.Format)
	}
	ext, ok := lockfile.ExtAdapterFor(project.IdentityYarn)
	if !ok {
		t.Fatal("missing yarn adapter")
	}
	g, _, err := ext.ReadWithExtensions(context.Background(), lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Packages) == 0 {
		t.Fatal("expected packages")
	}
	res, err := ext.(lockfile.PreservingEncoder).EncodePreserving(context.Background(), lockPath, g, data, nil, lockfile.Detection{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unchanged {
		t.Fatal("expected unchanged classic lock")
	}
}

func TestLockBridgeYarnBerryNMFixture(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, "fixtures", "locks", "yarn", "berry-nm")
	lockPath := filepath.Join(dir, "yarn.lock")
	ext, ok := lockfile.ExtAdapterFor(project.IdentityYarn)
	if !ok {
		t.Fatal("missing yarn adapter")
	}
	g, _, err := ext.ReadWithExtensions(context.Background(), lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Packages) == 0 {
		t.Fatal("expected packages")
	}
	data, _ := os.ReadFile(lockPath)
	if yarn.DetectVariant(data, dir) != yarn.VariantBerryNM {
		t.Fatalf("expected berry-nm variant")
	}
}

func TestLockBridgeYarnBerryPnPReadOnly(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, "fixtures", "locks", "yarn", "berry-pnp-readonly")
	lockPath := filepath.Join(dir, "yarn.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := berry.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	_, ext, err := berry.ToPnPGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !berry.IsPnPGraph(ext) {
		t.Fatal("expected pnp tag")
	}
	if yarn.DetectVariant(data, dir) != yarn.VariantBerryPnP {
		t.Fatal("expected berry-pnp variant")
	}
}

func TestLockBridgeYarnPnPInstallRejected(t *testing.T) {
	root := moduleRoot(t)
	fixture := filepath.Join(root, "fixtures", "locks", "yarn", "berry-pnp-readonly")
	proj := &project.Project{Root: fixture, Identity: project.IdentityYarn}
	err := gateYarnPnPInstallExported(proj)
	if err == nil {
		t.Fatal("expected PnP install rejection")
	}
	if apperr.CodeOf(err) != apperr.PNPUnsupported {
		t.Fatalf("code=%s want %s", apperr.CodeOf(err), apperr.PNPUnsupported)
	}
}

func gateYarnPnPInstallExported(proj *project.Project) error {
	lockPath := filepath.Join(proj.Root, "yarn.lock")
	prior, err := os.ReadFile(lockPath)
	if err != nil {
		return err
	}
	if !yarn.IsPnPProject(proj.Root, prior) {
		return nil
	}
	return apperr.New(apperr.PNPUnsupported, "install", "yarn.lock", "test gate")
}

func TestLockBridgeYarnIdentityAdapter(t *testing.T) {
	root := moduleRoot(t)
	lockPath := filepath.Join(root, "fixtures", "locks", "yarn", "classic-v1", "yarn.lock")
	ext, ok := lockfile.ExtAdapterFor(project.IdentityYarn)
	if !ok {
		t.Fatal("missing yarn adapter")
	}
	if _, err := ext.Read(context.Background(), lockPath); err != nil {
		t.Fatal(err)
	}
}

func TestYarnDetectVariants(t *testing.T) {
	root := moduleRoot(t)
	classicData, _ := os.ReadFile(filepath.Join(root, "fixtures", "locks", "yarn", "classic-v1", "yarn.lock"))
	if yarn.DetectVariant(classicData, "") != yarn.VariantClassic {
		t.Fatal("expected classic")
	}
	nmDir := filepath.Join(root, "fixtures", "locks", "yarn", "berry-nm")
	nmData, _ := os.ReadFile(filepath.Join(nmDir, "yarn.lock"))
	if yarn.DetectVariant(nmData, nmDir) != yarn.VariantBerryNM {
		t.Fatal("expected berry-nm")
	}
	pnpDir := filepath.Join(root, "fixtures", "locks", "yarn", "berry-pnp-readonly")
	pnpData, _ := os.ReadFile(filepath.Join(pnpDir, "yarn.lock"))
	if yarn.DetectVariant(pnpData, pnpDir) != yarn.VariantBerryPnP {
		t.Fatal("expected berry-pnp")
	}
	_ = bytes.Equal(classicData, classicData)
}
