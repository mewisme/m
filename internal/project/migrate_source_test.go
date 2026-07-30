package project_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/project"
)

func writeMigrateFixture(t *testing.T, dir string, pkg string, locks map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if pkg != "" {
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range locks {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMigratableLockCandidatesPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x"}`, map[string]string{
		"pnpm-lock.yaml":      "lock",
		"package-lock.json":   "lock",
		"npm-shrinkwrap.json": "lock",
	})
	cands := project.MigratableLockCandidates(dir)
	if len(cands) != 2 {
		t.Fatalf("got %d candidates: %+v", len(cands), cands)
	}
	if cands[0].File != "npm-shrinkwrap.json" || cands[1].File != "pnpm-lock.yaml" {
		t.Fatalf("order=%+v", cands)
	}
}

func TestResolveMigrateSourceManifestWinsOverStaleLocks(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x","packageManager":"pnpm@9.0.0"}`, map[string]string{
		"pnpm-lock.yaml":    "lock",
		"package-lock.json": "lock",
	})
	id, path, err := project.ResolveMigrateSource(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if id != project.IdentityPNPM || !strings.HasSuffix(path, "pnpm-lock.yaml") {
		t.Fatalf("id=%s path=%s", id, path)
	}
}

func TestResolveMigrateSourceSoleLock(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x"}`, map[string]string{
		"package-lock.json": "lock",
	})
	id, _, err := project.ResolveMigrateSource(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if id != project.IdentityNPM {
		t.Fatalf("id=%s", id)
	}
}

func TestResolveMigrateSourceShrinkwrapOverPackageLock(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x"}`, map[string]string{
		"package-lock.json":   "lock",
		"npm-shrinkwrap.json": "lock",
	})
	id, path, err := project.ResolveMigrateSource(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if id != project.IdentityNPM || !strings.HasSuffix(path, "npm-shrinkwrap.json") {
		t.Fatalf("id=%s path=%s", id, path)
	}
}

func TestResolveMigrateSourceMultiLockPicksPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x"}`, map[string]string{
		"pnpm-lock.yaml": "lock",
		"yarn.lock":      "lock",
	})
	id, path, err := project.ResolveMigrateSource(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if id != project.IdentityPNPM || !strings.HasSuffix(path, "pnpm-lock.yaml") {
		t.Fatalf("id=%s path=%s", id, path)
	}
}

func TestResolveMigrateSourceOnlyMLock(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x"}`, map[string]string{
		"m.lock": "{}",
	})
	_, _, err := project.ResolveMigrateSource(dir, "")
	if err == nil {
		t.Fatal("expected nothing to migrate")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "nothing to migrate") {
		t.Fatalf("msg=%v", err)
	}
}

func TestResolveMigrateSourceExplicitFromOverridesAmbiguity(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x"}`, map[string]string{
		"pnpm-lock.yaml": "lock",
		"yarn.lock":      "lock",
	})
	id, _, err := project.ResolveMigrateSource(dir, "pnpm")
	if err != nil {
		t.Fatal(err)
	}
	if id != project.IdentityPNPM {
		t.Fatalf("id=%s", id)
	}
}

func TestResolveMigrateSourceManifestConflict(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x","packageManager":"npm@10.0.0"}`, map[string]string{
		"pnpm-lock.yaml": "lock",
	})
	_, _, err := project.ResolveMigrateSource(dir, "")
	if err == nil {
		t.Fatal("expected conflict")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestResolveMigrateSourceMewManifestWithForeignLocks(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x","packageManager":"mew@0.1.0"}`, map[string]string{
		"m.lock":         "{}",
		"pnpm-lock.yaml": "lock",
	})
	_, _, err := project.ResolveMigrateSource(dir, "")
	if err == nil {
		t.Fatal("expected --from hint")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "--from") {
		t.Fatalf("msg=%v", err)
	}
}

func TestDetectIdentityForMigrateAllowsMultiLock(t *testing.T) {
	dir := t.TempDir()
	writeMigrateFixture(t, dir, `{"name":"x"}`, map[string]string{
		"pnpm-lock.yaml": "lock",
		"yarn.lock":      "lock",
	})
	p, err := project.DetectIdentityForMigrate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Identity != project.IdentityMew {
		t.Fatalf("identity=%s", p.Identity)
	}
}
