package project_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/testkit"
)

func TestDetectIdentityFixtures(t *testing.T) {
	root := testkit.ModuleRoot(t)
	cases := []struct {
		dir string
		id  project.Identity
	}{
		{"fixtures/identity/npm-lock", project.IdentityNPM},
		{"fixtures/identity/pnpm-lock", project.IdentityPNPM},
		{"fixtures/identity/nub-lock", project.IdentityNub},
		{"fixtures/identity/packageManager-field", project.IdentityPNPM},
		{"fixtures/identity/mew-native", project.IdentityMew},
	}
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			p, err := project.DetectIdentity(filepath.Join(root, filepath.FromSlash(c.dir)))
			if err != nil {
				t.Fatal(err)
			}
			if p.Identity != c.id {
				t.Fatalf("got %s want %s signals=%v", p.Identity, c.id, p.Signals)
			}
		})
	}
}

// A packageManager declaration that disagrees with the incumbent lockfile is
// advisory, not fatal: the lockfile is the authority and wins.
func TestDeclarationLosesToLockfile(t *testing.T) {
	root := testkit.ModuleRoot(t)
	p, err := project.DetectIdentity(filepath.Join(root, "fixtures", "identity", "conflict-signals"))
	if err != nil {
		t.Fatalf("declaration/lockfile mismatch must not fail: %v", err)
	}
	if p.Identity != project.IdentityPNPM {
		t.Fatalf("lockfile must win: got %s want %s signals=%v", p.Identity, project.IdentityPNPM, p.Signals)
	}
	if p.Declared != project.IdentityNPM {
		t.Fatalf("declared identity must be preserved: got %q want %q", p.Declared, project.IdentityNPM)
	}
}

// With no lockfile there is no authority to inherit, so the project is Mew
// native and install writes m.lock. The declaration is recorded, not obeyed.
func TestDeclarationWithoutLockfile(t *testing.T) {
	root := testkit.ModuleRoot(t)
	p, err := project.DetectIdentity(filepath.Join(root, "fixtures", "identity", "declared-yarn-no-lock"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Identity != project.IdentityMew {
		t.Fatalf("declaration must not confer authority: got %s signals=%v", p.Identity, p.Signals)
	}
	if p.Declared != project.IdentityYarn {
		t.Fatalf("declared identity must be preserved: got %q want %q", p.Declared, project.IdentityYarn)
	}
}

// No declaration and no lockfile is a native mew project.
func TestNoSignalsDefaultsToMew(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := project.DetectIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Identity != project.IdentityMew {
		t.Fatalf("got %s want mew", p.Identity)
	}
	if p.Declared != "" {
		t.Fatalf("declared must be empty: got %q", p.Declared)
	}
}

// Two lockfiles from different authorities is a genuine, unresolvable conflict.
func TestConflictingLockfiles(t *testing.T) {
	root := testkit.ModuleRoot(t)
	_, err := project.DetectIdentity(filepath.Join(root, "fixtures", "identity", "conflict-lockfiles"))
	if err == nil {
		t.Fatal("expected conflict")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("msg=%v", err)
	}
}

func TestFindRoot(t *testing.T) {
	root := testkit.ModuleRoot(t)
	npm := filepath.Join(root, "fixtures", "identity", "npm-lock")
	sub := filepath.Join(npm, "nested", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := project.FindRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != npm {
		t.Fatalf("got %s want %s", got, npm)
	}
}

func TestReadsBrandedConfig(t *testing.T) {
	if project.ReadsBrandedConfig(project.IdentityMew) {
		t.Fatal("mew must not read branded config")
	}
	if !project.ReadsBrandedConfig(project.IdentityNPM) {
		t.Fatal("npm may read branded config via adapters")
	}
}
