package charter

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// indexedMVPs lists every MVP from plans/INDEX.md that must map to a charter objective.
var indexedMVPs = []string{
	"0001", "0002", "0003", "0004", "0005", "0006", "0007", "0008", "0009",
	"0010", "0011", "0012", "0013", "0014", "0015", "0016", "0017", "0018", "0019",
	"0020", "0021", "0022", "0023", "0024", "0025", "0026", "0027", "0028", "0029",
	"0030", "0031",
	"0040", "0041", "0042", "0043", "0044", "0045", "0046",
	"0050", "0051", "0052", "0053", "0054", "0055", "0056", "0057",
	"0060", "0061", "0062",
	"0070", "0071", "0072", "0073", "0074",
	"0080", "0081", "0082", "0083", "0084", "0085", "0086", "0087", "0088", "0089",
	"0090",
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func readDoc(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func TestCharterNamesProductIdentity(t *testing.T) {
	charter := readDoc(t, repoRoot(t), filepath.Join("docs", "charter.md"))

	required := []string{"`m`", "`mx`", "`m.lock`", "Nub", "behavioral reference"}
	for _, term := range required {
		if !strings.Contains(charter, term) {
			t.Errorf("docs/charter.md missing %q", term)
		}
	}

	forbidden := []string{
		"source-level port",
		"transliterate",
		"line-by-line translation",
		"embed Nub",
	}
	for _, phrase := range forbidden {
		if strings.Contains(strings.ToLower(charter), strings.ToLower(phrase)) {
			// charter.md uses "not a source-level port target" — allow negated forms only
			if strings.Contains(charter, "not a source-level port") {
				continue
			}
			t.Errorf("docs/charter.md contains discouraged port language: %q", phrase)
		}
	}
}

func TestCompatibilityAxesCoverAllAxes(t *testing.T) {
	axes := readDoc(t, repoRoot(t), filepath.Join("docs", "compatibility-axes.md"))

	for _, axis := range []string{"CLI", "Lockfile", "Config", "Runtime", "Layout"} {
		if !strings.Contains(axes, axis) {
			t.Errorf("docs/compatibility-axes.md missing axis %q", axis)
		}
	}

	states := []string{"parity", "intentional divergence", "extension", "deferred"}
	for _, state := range states {
		if !strings.Contains(axes, state) {
			t.Errorf("docs/compatibility-axes.md missing compatibility state %q", state)
		}
	}
}

func TestEveryIndexMVPMapped(t *testing.T) {
	axes := readDoc(t, repoRoot(t), filepath.Join("docs", "compatibility-axes.md"))

	for _, id := range indexedMVPs {
		// Table rows start with "| 00xx |"
		pattern := regexp.MustCompile(`\|\s*` + id + `\s*\|`)
		if !pattern.MatchString(axes) {
			t.Errorf("MVP %s not found in docs/compatibility-axes.md INDEX map", id)
		}
	}
}

func TestDirectScriptShortcutsAreExtension(t *testing.T) {
	charter := readDoc(t, repoRoot(t), filepath.Join("docs", "charter.md"))
	axes := readDoc(t, repoRoot(t), filepath.Join("docs", "compatibility-axes.md"))

	if !strings.Contains(charter, "m dev") || !strings.Contains(charter, "Direct script shortcuts") {
		t.Error("docs/charter.md must document direct script shortcuts")
	}
	if !strings.Contains(axes, "extension") || !strings.Contains(axes, "0042") {
		t.Error("docs/compatibility-axes.md must list direct shortcuts as extension (0042)")
	}
}

func TestADRProcessDocumented(t *testing.T) {
	root := repoRoot(t)
	adrREADME := readDoc(t, root, filepath.Join("docs", "adr", "README.md"))
	template := readDoc(t, root, filepath.Join("docs", "adr", "0000-template.md"))

	if !strings.Contains(adrREADME, "before") {
		t.Error("docs/adr/README.md must require ADRs before irreversible decisions")
	}
	for _, section := range []string{"## Context", "## Decision", "## Consequences", "## Rollback"} {
		if !strings.Contains(template, section) {
			t.Errorf("ADR template missing section %q", section)
		}
	}
}

func TestNamingConventionsFrozen(t *testing.T) {
	naming := readDoc(t, repoRoot(t), filepath.Join("docs", "naming.md"))

	for _, term := range []string{"`m`", "`mx`", "`m.lock`", "`nub.lock`", "ERR_M_", "MEW_"} {
		if !strings.Contains(naming, term) {
			t.Errorf("docs/naming.md missing %q", term)
		}
	}
}

func TestCharterFixturesExist(t *testing.T) {
	root := repoRoot(t)
	fixtures := []struct {
		dir  string
		file string
	}{
		{"fixtures/charter/npm-app", "package-lock.json"},
		{"fixtures/charter/pnpm-app", "pnpm-lock.yaml"},
		{"fixtures/charter/nub-app", "nub.lock"},
		{"fixtures/charter/empty", ".gitkeep"},
	}
	for _, f := range fixtures {
		path := filepath.Join(root, f.dir, f.file)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing fixture file %s: %v", path, err)
		}
	}
}

func TestMigrationReferencesFixtures(t *testing.T) {
	migration := readDoc(t, repoRoot(t), filepath.Join("docs", "migration.md"))

	for _, fixture := range []string{"npm-app", "pnpm-app", "nub-app", "empty"} {
		if !strings.Contains(migration, fixture) {
			t.Errorf("docs/migration.md missing fixture reference %q", fixture)
		}
	}
}

func TestLockfilePreservationPolicy(t *testing.T) {
	charter := readDoc(t, repoRoot(t), filepath.Join("docs", "charter.md"))

	for _, lock := range []string{"package-lock.json", "pnpm-lock.yaml", "nub.lock", "m.lock"} {
		if !strings.Contains(charter, lock) {
			t.Errorf("docs/charter.md missing lockfile %q in preservation policy", lock)
		}
	}
}
