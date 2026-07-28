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

func TestConflictSignals(t *testing.T) {
	root := testkit.ModuleRoot(t)
	_, err := project.DetectIdentity(filepath.Join(root, "fixtures", "identity", "conflict-signals"))
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
