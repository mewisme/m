package lockfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	_ "github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/lockfile"
)

func readFixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "fixtures", "locks"}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDetectPnpmPackageManager(t *testing.T) {
	data := readFixture(t, "pnpm", "v9", "pnpm-lock.yaml")
	det, err := lockfile.DetectPnpmWithContext(data, lockfile.ProjectHints{
		PackageManager: "pnpm@10.4.0",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if det.Format != "pnpm-v10" || det.ProducerMajor != 10 {
		t.Fatalf("got %+v", det)
	}
	if det.Confidence != lockfile.DetectionCertain {
		t.Fatalf("confidence=%s", det.Confidence)
	}
}

func TestDetectPnpmDevEngines(t *testing.T) {
	data := readFixture(t, "pnpm", "v9", "pnpm-lock.yaml")
	det, err := lockfile.DetectPnpmWithContext(data, lockfile.ProjectHints{
		DevEnginesPM: "pnpm@11.2.0",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if det.ProducerMajor != 11 {
		t.Fatalf("got %+v", det)
	}
}

func TestDetectPnpmExplicitMajor(t *testing.T) {
	data := readFixture(t, "pnpm", "v9", "pnpm-lock.yaml")
	det, err := lockfile.DetectPnpmWithContext(data, lockfile.ProjectHints{}, 9)
	if err != nil {
		t.Fatal(err)
	}
	if !det.ExplicitMajor || det.ProducerMajor != 9 {
		t.Fatalf("got %+v", det)
	}
}

func TestDetectPnpmConflict(t *testing.T) {
	data := readFixture(t, "pnpm", "v9", "pnpm-lock.yaml")
	_, err := lockfile.DetectPnpmWithContext(data, lockfile.ProjectHints{
		PackageManager: "pnpm@9.0.0",
	}, 10)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if _, ok := err.(*lockfile.DetectionConflictError); !ok {
		t.Fatalf("got %T", err)
	}
}

func TestDetectPnpmAmbiguousWithoutHints(t *testing.T) {
	data := readFixture(t, "pnpm", "v9", "pnpm-lock.yaml")
	det, err := lockfile.DetectPnpm(data)
	if err != nil {
		t.Fatal(err)
	}
	if det.ProducerMajor != 0 {
		t.Fatalf("ProducerMajor=%d want 0 for ambiguous v9-shaped lock", det.ProducerMajor)
	}
	if det.Certified() {
		t.Fatal("v9-shaped lock without markers must not be certified")
	}
}

func TestDetectPnpmV10StructuralEvidence(t *testing.T) {
	data := readFixture(t, "pnpm", "v10", "pnpm-lock.yaml")
	det, err := lockfile.DetectPnpm(data)
	if err != nil {
		t.Fatal(err)
	}
	if det.Format != "pnpm-v10" {
		t.Fatalf("format=%s", det.Format)
	}
}

func TestDetectPnpmV11StructuralEvidence(t *testing.T) {
	data := readFixture(t, "pnpm", "v11", "pnpm-lock.yaml")
	det, err := lockfile.DetectPnpm(data)
	if err != nil {
		t.Fatal(err)
	}
	if det.Format != "pnpm-v11" {
		t.Fatalf("format=%s", det.Format)
	}
}

func TestDetectPnpmRejectsLegacyV6(t *testing.T) {
	data := readFixture(t, "pnpm", "unsupported", "v6", "pnpm-lock.yaml")
	_, err := lockfile.DetectPnpm(data)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if apperr.CodeOf(err) != apperr.LockUnsupported {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}
