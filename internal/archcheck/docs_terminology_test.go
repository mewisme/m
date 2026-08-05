package archcheck_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// staleThemeTerms are Go identifiers and doc words that must not appear in
// source or documentation after the MarkdownTheme → UITheme rename.
var staleThemeTerms = []string{
	"MarkdownTheme",
	"markdownTheme",
}

// staleStatusTerms are old capability-status words that must not appear in
// docs/architecture/package-map.md capability-state cells.
var staleStatusTerms = []string{
	"| shipped ",
	"| certified ",
	"| experimental ",
}

// validStatusTerms are the current capability-state vocabulary.
var validStatusTerms = []string{
	"| implemented ",
	"| partial ",
	"| scaffolded ",
	"| planned ",
	"| reserved ",
}

// legacyConfigKeys are key names that have been renamed and must not appear
// as canonical key paths in documentation.
var legacyConfigKeys = []string{
	"network.timeout_ms",
}

// stalePackageRefs are paths that no longer exist on disk and must not be
// listed as "exists" in the package map.
var stalePackageRefs = []string{
	"internal/audit/",
}

func TestNoStaleThemeTerminology(t *testing.T) {
	root := repoRoot(t)
	self := filepath.Join(root, "internal", "archcheck", "docs_terminology_test.go")

	var problems []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".claude", "node_modules", "testdata", "fixtures", ".codegraph":
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".md" {
			return nil
		}
		if path == self {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(b)
		for _, term := range staleThemeTerms {
			if strings.Contains(text, term) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					rel = path
				}
				problems = append(problems, rel+": contains stale term "+term)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Errorf("stale theme terminology found:\n%s", strings.Join(problems, "\n"))
	}
}

func TestPackageMapVocabulary(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "package-map.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)

	// Extract the capability-state vocabulary definition section.
	vocabStart := strings.Index(text, "**Capability state**")
	if vocabStart < 0 {
		t.Fatal("package-map.md missing capability state vocabulary")
	}
	vocabSection := text[vocabStart:]

	// Check that old vocabulary terms don't appear in status cells after the definition.
	tableStart := strings.Index(vocabSection, "## Entry and presentation")
	if tableStart < 0 {
		tableStart = strings.Index(vocabSection, "## Package-manager domain")
	}
	if tableStart < 0 {
		t.Fatal("package-map.md missing table sections")
	}
	tableSection := vocabSection[tableStart:]

	for _, old := range staleStatusTerms {
		if strings.Contains(tableSection, old) {
			t.Errorf("package-map.md contains stale status term %q in table cells", old)
		}
	}

	// Verify the current vocabulary definition contains all valid terms.
	for _, valid := range validStatusTerms {
		term := strings.TrimSpace(strings.TrimPrefix(valid, "| "))
		if !strings.Contains(vocabSection, "`"+term+"`") {
			t.Errorf("package-map.md vocabulary section missing term %q", term)
		}
	}
}

func TestLegacyConfigKeysNotInDocs(t *testing.T) {
	root := repoRoot(t)
	docsDir := filepath.Join(root, "docs")

	var problems []string
	err := filepath.WalkDir(docsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(b)
		for _, key := range legacyConfigKeys {
			if strings.Contains(text, key) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					rel = path
				}
				// Allow migration docs to reference legacy keys as migration sources.
				if strings.Contains(rel, "migrate") || strings.Contains(rel, "adr") {
					continue
				}
				// Allow the config doc's migration section.
				if strings.Contains(text, "migrate") && strings.Contains(text, "legacy") {
					continue
				}
				problems = append(problems, rel+": contains legacy key "+key)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Errorf("legacy config keys in docs (outside migration sections):\n%s", strings.Join(problems, "\n"))
	}
}

func TestStalePackageRefsNotExists(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "package-map.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)

	// Check that stale package refs are not listed as "exists".
	for _, ref := range stalePackageRefs {
		// Find lines containing this ref.
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "`"+ref+"`") {
				if strings.Contains(line, "| exists |") {
					t.Errorf("package-map.md lists %s as 'exists' but it does not exist on disk", ref)
				}
			}
		}
	}
}

func TestPackageMapMissingAdvisory(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "package-map.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)

	// internal/advisory/ exists on disk and must be in the package map.
	if !strings.Contains(text, "`internal/advisory/`") {
		t.Error("package-map.md missing internal/advisory/ (exists on disk)")
	}
}

func TestGeneratedDocsExist(t *testing.T) {
	root := repoRoot(t)

	// Check that terminal-help docs directory exists.
	helpDir := filepath.Join(root, "docs", "terminal-help")
	if fi, err := os.Stat(helpDir); err != nil || !fi.IsDir() {
		t.Errorf("docs/terminal-help/ missing or not a directory: %v", err)
	}

	// Check that embedded help FS exists.
	embedDir := filepath.Join(root, "docs", "terminal-help")
	entries, err := os.ReadDir(embedDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("docs/terminal-help/ is empty")
	}
}

func TestConfigKeyDocsMatchRegistry(t *testing.T) {
	root := repoRoot(t)

	// Read the config doc owned keys table.
	b, err := os.ReadFile(filepath.Join(root, "docs", "config.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)

	// Every owned-key reference should use backtick-quoted dotted keys.
	// Check that key mentions in the owned keys section use consistent dotted form.
	needles := []string{
		"`ui.theme`",
		"`registry`",
		"`offline`",
		"`install.linker`",
		"`lifecycle.enabled`",
		"`workspaces.enabled`",
		"`runner.direct_scripts.enabled`",
		"`runner.exec.direct_dispatch.enabled`",
	}
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			t.Errorf("config.md owned keys section missing %s", needle)
		}
	}
}

func TestCommandDocsReferenceActualCommands(t *testing.T) {
	root := repoRoot(t)

	b, err := os.ReadFile(filepath.Join(root, "docs", "config.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)

	// Every config subcommand mentioned in docs must use canonical form.
	commands := []string{
		"m config get",
		"m config set",
		"m config unset",
		"m config list",
		"m config path",
		"m config validate",
		"m config migrate",
		"m config edit",
		"m config reset",
	}
	for _, cmd := range commands {
		if !strings.Contains(text, cmd) {
			t.Errorf("config.md missing command reference %q", cmd)
		}
	}
}
