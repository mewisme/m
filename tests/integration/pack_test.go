package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/pack"
	"github.com/mewisme/mew/internal/testkit"
)

func TestPackMinimalFixtureGoldenFileList(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "pack/minimal-package", projDir)

	goldenPath := filepath.Join(testkit.ModuleRoot(t), "testdata", "pack", "minimal-package-files.json")
	wantRaw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if err := json.Unmarshal(wantRaw, &want); err != nil {
		t.Fatal(err)
	}

	pkgJSON, err := os.ReadFile(filepath.Join(projDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := pack.ListFiles(projDir, pkgJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("files: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("files[%d]: got %q want %q (full %v)", i, got[i], want[i], got)
		}
	}

	dest := filepath.Join(t.TempDir(), "out")
	res, err := pack.Pack(t.Context(), pack.Options{Root: projDir, PackDestination: dest})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(res.TarballPath) != "minimal-pack-fixture-1.2.3.tgz" {
		t.Fatalf("tarball %q", res.TarballPath)
	}
	if _, err := os.Stat(res.TarballPath); err != nil {
		t.Fatal(err)
	}
}

func TestPackCLI(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "pack/minimal-package", projDir)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "pack", "--pack-destination", projDir)
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "minimal-pack-fixture-1.2.3.tgz")); err != nil {
		t.Fatal(err)
	}
}
