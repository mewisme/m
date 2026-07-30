package bootstrap_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestCleanCloneGates(t *testing.T) {
	root := testkit.ModuleRoot(t)

	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), "go 1.26.5") {
		t.Fatalf("go.mod missing go 1.26.5:\n%s", mod)
	}

	license, err := os.ReadFile(filepath.Join(root, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	licenseText := string(license)
	if !strings.Contains(licenseText, "Apache License") || !strings.Contains(licenseText, "Version 2.0") {
		t.Fatal("LICENSE must be Apache License 2.0")
	}

	notice, err := os.ReadFile(filepath.Join(root, "NOTICE"))
	if err != nil {
		t.Fatal(err)
	}
	noticeText := string(notice)
	if !strings.Contains(noticeText, "Copyright 2026 Nguyễn Mậu Minh") {
		t.Fatal("NOTICE must contain copyright attribution")
	}
	if !strings.Contains(noticeText, "Apache License, Version 2.0") {
		t.Fatal("NOTICE must declare Apache-2.0")
	}

	mk, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	mkText := string(mk)
	for _, target := range []string{"test:", "vet:", "lint:", "race:", "fuzz-smoke:", "vuln:", "build:", "allowlist:"} {
		if !strings.Contains(mkText, target) {
			t.Errorf("Makefile missing target %q", target)
		}
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	outDir := t.TempDir()
	for _, pair := range []struct {
		pkg string
		bin string
	}{
		{"./cmd/m", "m" + ext},
		{"./cmd/mx", "mx" + ext},
	} {
		cmd := exec.Command("go", "build", "-o", filepath.Join(outDir, pair.bin), pair.pkg)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", pair.pkg, err, out)
		}
	}
}
