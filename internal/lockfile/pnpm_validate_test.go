package lockfile_test

import (
	"testing"

	"github.com/mewisme/mew/internal/lockfile"
)

func TestValidatePnpmProducerMajor(t *testing.T) {
	for _, major := range []int{9, 10, 11, 0} {
		if err := lockfile.ValidatePnpmProducerMajor(major); err != nil {
			t.Fatalf("major=%d: %v", major, err)
		}
	}
	for _, major := range []int{6, 8, 12} {
		if err := lockfile.ValidatePnpmProducerMajor(major); err == nil {
			t.Fatalf("major=%d should fail", major)
		}
	}
}

func TestValidatePnpmHintsRejectsUnsupportedField(t *testing.T) {
	err := lockfile.ValidatePnpmHints(lockfile.ProjectHints{
		PackageManager: "pnpm@8.15.0",
	}, 0)
	if err == nil {
		t.Fatal("expected pnpm@8 rejection")
	}
}

func TestValidatePnpmHintsRejectsRange(t *testing.T) {
	err := lockfile.ValidatePnpmHints(lockfile.ProjectHints{
		PackageManager: "pnpm@^9.0.0",
	}, 0)
	if err == nil {
		t.Fatal("expected range rejection")
	}
}

func TestDetectPnpmForProjectConflict(t *testing.T) {
	data := readFixture(t, "pnpm", "v9", "pnpm-lock.yaml")
	_, err := lockfile.DetectPnpmForProject(data, lockfile.ProjectHints{
		PackageManager: "pnpm@10.0.0",
	}, 9)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if _, ok := err.(*lockfile.DetectionConflictError); !ok {
		t.Fatalf("got %T", err)
	}
}
