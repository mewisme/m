package lockfile_test

import (
	"testing"

	"github.com/mewisme/mew/internal/lockfile"
)

func TestParsePMDeclarationBarePnpm(t *testing.T) {
	decl, err := lockfile.ParsePMDeclaration("packageManager", "pnpm")
	if err != nil {
		t.Fatal(err)
	}
	if decl.ProducerMajor != 0 || decl.EvidenceState != lockfile.PMEvidenceNone {
		t.Fatalf("got %+v", decl)
	}
}

func TestParsePMDeclarationExactMajor(t *testing.T) {
	decl, err := lockfile.ParsePMDeclaration("packageManager", "pnpm@9.15.9")
	if err != nil {
		t.Fatal(err)
	}
	if decl.ProducerMajor != 9 || decl.ExactVersion != "9.15.9" {
		t.Fatalf("got %+v", decl)
	}
}

func TestParsePMDeclarationRejectsLatest(t *testing.T) {
	_, err := lockfile.ParsePMDeclaration("packageManager", "pnpm@latest")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePMDeclarationRejectsRange(t *testing.T) {
	_, err := lockfile.ParsePMDeclaration("packageManager", "pnpm@^10")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePMDeclarationRejectsUnsupportedMajor(t *testing.T) {
	_, err := lockfile.ParsePMDeclaration("packageManager", "pnpm@8.15.0")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = lockfile.ParsePMDeclaration("packageManager", "pnpm@12")
	if err == nil {
		t.Fatal("expected pnpm@12 error")
	}
}

func TestDetectPnpmBarePackageManagerNoMajor(t *testing.T) {
	data := readFixture(t, "pnpm", "v9", "pnpm-lock.yaml")
	det, err := lockfile.DetectPnpmWithContext(data, lockfile.ProjectHints{
		PackageManager: "pnpm",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if det.ProducerMajor != 0 {
		t.Fatalf("ProducerMajor=%d want 0", det.ProducerMajor)
	}
}
