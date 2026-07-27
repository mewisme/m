package store_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/testkit"
)

const indexProcEnv = "MEW_STORE_INDEX_PROC"

type indexImportSpec struct {
	tgz       string
	integrity string
}

func TestIndexProcConcurrentDistinctImports(t *testing.T) {
	if role := os.Getenv(indexProcEnv); role != "" {
		runIndexProcChild(t, role)
		return
	}

	root := filepath.Join(t.TempDir(), "store")
	fixtureRoot := testkit.FixtureDir(t, "registry/v1")
	specs := []indexImportSpec{
		{
			tgz:       filepath.Join(fixtureRoot, "tarballs", "lodash-4.17.21.tgz"),
			integrity: "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63",
		},
		{
			tgz:       filepath.Join(fixtureRoot, "tarballs", "pkg-cli-1.0.0.tgz"),
			integrity: "sha256-6ffb2697417ee0f02ad400c8d92c46cfb5889cf84603cd1f797146fde316b5d0",
		},
		{
			tgz:       filepath.Join(fixtureRoot, "tarballs", "pkg-a-1.0.0.tgz"),
			integrity: "sha256-2e1afab8b566a6ac1019ae2ba9201ea8a036b0ca1463ed2b22673d4cc87b2354",
		},
	}

	const workers = 3
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		spec := specs[i]
		cmd := exec.Command(os.Args[0], "-test.run=^TestIndexProcConcurrentDistinctImports$", "-test.count=1")
		cmd.Env = append(os.Environ(),
			indexProcEnv+"=import",
			"MEW_STORE_ROOT="+root,
			"MEW_STORE_TGZ="+spec.tgz,
			"MEW_STORE_INTEGRITY="+spec.integrity,
		)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		go func(c *exec.Cmd) { errCh <- c.Wait() }(cmd)
	}
	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}

	ps := store.NewPackageStore(root)
	idx, err := store.ReadIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Packages) != workers {
		t.Fatalf("index entries=%d want %d", len(idx.Packages), workers)
	}
	keys, err := ps.ListPackageKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != workers {
		t.Fatalf("package keys=%d want %d", len(keys), workers)
	}
	count, _, err := ps.Status()
	if err != nil {
		t.Fatal(err)
	}
	if count != workers {
		t.Fatalf("status count=%d want %d", count, workers)
	}
}

func runIndexProcChild(t *testing.T, role string) {
	t.Helper()
	if role != "import" {
		t.Fatalf("unknown role %q", role)
	}
	root := os.Getenv("MEW_STORE_ROOT")
	tgz := os.Getenv("MEW_STORE_TGZ")
	integrity := os.Getenv("MEW_STORE_INTEGRITY")
	if root == "" || tgz == "" || integrity == "" {
		t.Fatal("missing child env")
	}
	ps := store.NewPackageStore(root)
	if _, err := importIntegrity(context.Background(), ps, tgz, integrity); err != nil {
		t.Fatal(err)
	}
}
