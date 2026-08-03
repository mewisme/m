package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/project"
)

func writeIdentityFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func hasSignalKind(p *project.Project, kind string) bool {
	for _, s := range p.Signals {
		if s.Kind == kind {
			return true
		}
	}
	return false
}

// Lockfiles are the only authority: a declaration alone yields mew, so install
// writes m.lock rather than adopting the declared manager.
func TestDeclarationWithoutLockfileIsMewAuthority(t *testing.T) {
	dir := writeIdentityFiles(t, map[string]string{
		"package.json": `{"name":"x","version":"1.0.0","packageManager":"pnpm@9"}`,
	})
	p, err := project.DetectIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Identity != project.IdentityMew {
		t.Fatalf("identity: got %q want %q signals=%v", p.Identity, project.IdentityMew, p.Signals)
	}
	if p.Declared != project.IdentityPNPM {
		t.Fatalf("declared: got %q want %q", p.Declared, project.IdentityPNPM)
	}
	if !hasSignalKind(p, "packageManager") {
		t.Fatalf("missing packageManager signal: %v", p.Signals)
	}
	if !hasSignalKind(p, "default") {
		t.Fatalf("missing default signal: %v", p.Signals)
	}
}

// The incumbent lockfile outranks a disagreeing declaration without erroring.
func TestIncumbentLockfileOutranksDeclaration(t *testing.T) {
	dir := writeIdentityFiles(t, map[string]string{
		"package.json":      `{"name":"x","version":"1.0.0","packageManager":"pnpm@9"}`,
		"package-lock.json": `{"lockfileVersion":3}`,
	})
	p, err := project.DetectIdentity(dir)
	if err != nil {
		t.Fatalf("declaration/lockfile mismatch must not fail: %v", err)
	}
	if p.Identity != project.IdentityNPM {
		t.Fatalf("identity: got %q want %q signals=%v", p.Identity, project.IdentityNPM, p.Signals)
	}
	if p.Declared != project.IdentityPNPM {
		t.Fatalf("declared: got %q want %q", p.Declared, project.IdentityPNPM)
	}
}

// No declaration and no lockfile is a native mew project.
func TestNoDeclarationNoLockfileIsMew(t *testing.T) {
	dir := writeIdentityFiles(t, map[string]string{
		"package.json": `{"name":"x","version":"1.0.0"}`,
	})
	p, err := project.DetectIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Identity != project.IdentityMew {
		t.Fatalf("identity: got %q want %q", p.Identity, project.IdentityMew)
	}
	if p.Declared != "" {
		t.Fatalf("declared: got %q want empty", p.Declared)
	}
}

// Two lockfiles from different authorities is unresolvable.
func TestConflictingLockfilesFail(t *testing.T) {
	dir := writeIdentityFiles(t, map[string]string{
		"package.json":      `{"name":"x","version":"1.0.0"}`,
		"package-lock.json": `{"lockfileVersion":3}`,
		"pnpm-lock.yaml":    "lockfileVersion: '9.0'\n",
	})
	_, err := project.DetectIdentity(dir)
	if err == nil {
		t.Fatal("expected conflicting lockfile error")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code: got %s want %s", apperr.CodeOf(err), apperr.Config)
	}
}

// devEngines.packageManager is a declaration signal with the same weight.
func TestDevEnginesDeclarationWithoutLockfileIsMew(t *testing.T) {
	dir := writeIdentityFiles(t, map[string]string{
		"package.json": `{"name":"x","version":"1.0.0","devEngines":{"packageManager":"yarn@4"}}`,
	})
	p, err := project.DetectIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Identity != project.IdentityMew {
		t.Fatalf("identity: got %q want %q signals=%v", p.Identity, project.IdentityMew, p.Signals)
	}
	if p.Declared != project.IdentityYarn {
		t.Fatalf("declared: got %q want %q", p.Declared, project.IdentityYarn)
	}
	if !hasSignalKind(p, "devEngines") {
		t.Fatalf("missing devEngines signal: %v", p.Signals)
	}
}
