package archcheck_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestProductionBuildNoCGO(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	env := append(os.Environ(), "CGO_ENABLED=0")
	for _, target := range []string{"./cmd/m", "./cmd/mx"} {
		out := filepath.Join(tmp, filepath.Base(target))
		cmd := exec.Command("go", "build", "-o", out, target)
		cmd.Dir = root
		cmd.Env = env
		if outBytes, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("CGO_ENABLED=0 build %s: %v\n%s", target, err, outBytes)
		}
	}
}

func TestProductionVetNoCGO(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("go", "vet", "./cmd/...", "./internal/...")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("CGO_ENABLED=0 vet: %v\n%s", err, out)
	}
}
