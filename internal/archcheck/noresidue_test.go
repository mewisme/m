package archcheck_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Debug residue that must never reach production source again: an absolute
// developer-machine path, a session-scoped debug logger, and the editor region
// markers that wrapped them. These wrote a log file outside the product paths
// from inside ordinary command flow, which is exactly what must not happen.
var forbiddenResidue = []string{
	"debug-d57042",
	"agentIdentityDebugLog",
	"agentMigrateDebugLog",
	"#region agent log",
}

func TestProductionHasNoDebugResidue(t *testing.T) {
	root := repoRoot(t)

	// This test file necessarily contains the very strings it bans.
	self := filepath.Join(root, "internal", "archcheck", "noresidue_test.go")

	var problems []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			// Skip VCS internals and scratch worktrees: only committed
			// first-party source is in scope here.
			case ".git", ".claude", "node_modules", "testdata", "fixtures":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || path == self {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(b)
		for _, bad := range forbiddenResidue {
			if strings.Contains(text, bad) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					rel = path
				}
				problems = append(problems, rel+": contains "+bad)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("debug residue in production source:\n%s", strings.Join(problems, "\n"))
	}
}

// An absolute path rooted at a developer machine cannot work anywhere else, so
// no production Go source may hardcode one. Test files are exempt: they legitimately
// use Windows path literals as sample data for formatting and error-message
// assertions, where the string is never opened.
func TestNoDeveloperMachinePaths(t *testing.T) {
	root := repoRoot(t)

	var problems []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".claude", "node_modules", "testdata", "fixtures":
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lower := strings.ToLower(string(b))
		for _, bad := range []string{`f:\project`, `c:\users\`} {
			if strings.Contains(lower, bad) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					rel = path
				}
				problems = append(problems, rel+": hardcodes "+bad)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("developer-machine paths in source:\n%s", strings.Join(problems, "\n"))
	}
}
