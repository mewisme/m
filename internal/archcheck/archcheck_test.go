package archcheck_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// agentsPackages are paths from AGENTS.md repository shape that must appear
// in docs/architecture/package-map.md.
var agentsPackages = []string{
	"cmd/m/",
	"cmd/mx/",
	"internal/cli/",
	"internal/presentation/",
	"internal/app/",
	"internal/config/",
	"internal/manifest/",
	"internal/project/",
	"internal/workspace/",
	"internal/registry/",
	"internal/resolver/",
	"internal/lockfile/",
	"internal/fetch/",
	"internal/archive/",
	"internal/store/",
	"internal/linker/",
	"internal/transaction/",
	"internal/lifecycle/",
	"internal/policy/",
	"internal/runner/",
	"internal/process/",
	"internal/runtime/",
	"internal/transform/",
	"internal/node/",
	"internal/pmmanager/",
	"internal/compat/",
	"internal/testkit/",
	"runtime/",
	"tests/",
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

func TestPackageMapCoversAGENTS(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "package-map.md"))
	if err != nil {
		t.Fatal(err)
	}
	mapText := string(b)
	const tick = "`"
	for _, p := range agentsPackages {
		// Allow with or without trailing slash in the map.
		trimmed := strings.TrimSuffix(p, "/")
		if !strings.Contains(mapText, tick+p+tick) &&
			!strings.Contains(mapText, tick+trimmed+tick) &&
			!strings.Contains(mapText, tick+trimmed+"/"+tick) {
			t.Errorf("package-map.md missing AGENTS path %q", p)
		}
	}
}

func TestForbiddenImportsDocExists(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "docs", "architecture", "forbidden-imports.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, needle := range []string{"cmd/m", "internal/cli", "internal/resolver", "transaction"} {
		if !strings.Contains(text, needle) {
			t.Errorf("forbidden-imports.md missing %q", needle)
		}
	}
}

type listPkg struct {
	ImportPath string
	Imports    []string
	Dir        string
}

func goList(t *testing.T, root string) []listPkg {
	t.Helper()
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list: %v\n%s", err, ee.Stderr)
		}
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var pkgs []listPkg
	for dec.More() {
		var p listPkg
		if err := dec.Decode(&p); err != nil {
			t.Fatal(err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

func TestForbiddenImportEdges(t *testing.T) {
	root := repoRoot(t)
	pkgs := goList(t, root)
	byPath := make(map[string]listPkg, len(pkgs))
	for _, p := range pkgs {
		byPath[p.ImportPath] = p
	}

	allowedCmd := map[string]bool{
		"github.com/mewisme/mew/internal/app": true,
		"github.com/mewisme/mew/internal/cli": true,
	}

	for _, cmdPath := range []string{"github.com/mewisme/mew/cmd/m", "github.com/mewisme/mew/cmd/mx"} {
		p, ok := byPath[cmdPath]
		if !ok {
			t.Fatalf("missing package %s", cmdPath)
		}
		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, "github.com/mewisme/mew/") {
				continue
			}
			if !allowedCmd[imp] {
				t.Errorf("%s must not import %s (only internal/app and internal/cli)", cmdPath, imp)
			}
		}
	}

	cliForbidden := []string{
		"github.com/mewisme/mew/internal/linker",
		"github.com/mewisme/mew/internal/store",
		"github.com/mewisme/mew/internal/fetch",
	}
	cli := byPath["github.com/mewisme/mew/internal/cli"]
	for _, bad := range cliForbidden {
		for _, imp := range cli.Imports {
			if imp == bad || strings.HasPrefix(imp, bad+"/") {
				t.Errorf("internal/cli must not import %s", imp)
			}
		}
	}

	resolverForbidden := []string{
		"github.com/mewisme/mew/internal/linker",
		"github.com/mewisme/mew/internal/transaction",
		"github.com/mewisme/mew/internal/runner",
		"github.com/mewisme/mew/internal/fetch",
		"github.com/mewisme/mew/internal/store",
	}
	res := byPath["github.com/mewisme/mew/internal/resolver"]
	for _, bad := range resolverForbidden {
		for _, imp := range res.Imports {
			if imp == bad || strings.HasPrefix(imp, bad+"/") {
				t.Errorf("internal/resolver must not import %s", imp)
			}
		}
	}

	diagForbidden := []string{
		"github.com/mewisme/mew/internal/registry",
		"github.com/mewisme/mew/internal/fetch",
		"github.com/mewisme/mew/internal/linker",
	}
	for _, pkgPath := range []string{
		"github.com/mewisme/mew/internal/apperr",
		"github.com/mewisme/mew/internal/diagnostics",
		"github.com/mewisme/mew/internal/trace",
	} {
		p, ok := byPath[pkgPath]
		if !ok {
			t.Fatalf("missing package %s", pkgPath)
		}
		for _, bad := range diagForbidden {
			for _, imp := range p.Imports {
				if imp == bad || strings.HasPrefix(imp, bad+"/") {
					t.Errorf("%s must not import %s", pkgPath, imp)
				}
			}
		}
	}

	cfgForbidden := []string{
		"github.com/mewisme/mew/internal/resolver",
		"github.com/mewisme/mew/internal/linker",
		"github.com/mewisme/mew/internal/fetch",
	}
	for _, pkgPath := range []string{"github.com/mewisme/mew/internal/config", "github.com/mewisme/mew/internal/project"} {
		p, ok := byPath[pkgPath]
		if !ok {
			t.Fatalf("missing package %s", pkgPath)
		}
		for _, bad := range cfgForbidden {
			for _, imp := range p.Imports {
				if imp == bad || strings.HasPrefix(imp, bad+"/") {
					t.Errorf("%s must not import %s", pkgPath, imp)
				}
			}
		}
	}

	// Data-model packages stay free of fetch/linker/registry (registry itself exempt).
	modelForbidden := []string{
		"github.com/mewisme/mew/internal/fetch",
		"github.com/mewisme/mew/internal/linker",
		"github.com/mewisme/mew/internal/registry",
	}
	for _, pkgPath := range []string{
		"github.com/mewisme/mew/internal/graph",
		"github.com/mewisme/mew/internal/plan",
		"github.com/mewisme/mew/internal/snapshot",
		"github.com/mewisme/mew/internal/manifest",
		"github.com/mewisme/mew/internal/policy",
		"github.com/mewisme/mew/internal/capsule",
	} {
		p, ok := byPath[pkgPath]
		if !ok {
			t.Fatalf("missing package %s", pkgPath)
		}
		for _, bad := range modelForbidden {
			for _, imp := range p.Imports {
				if imp == bad || strings.HasPrefix(imp, bad+"/") {
					t.Errorf("%s must not import %s", pkgPath, imp)
				}
			}
		}
	}

	domainNoPresentation := []string{
		"github.com/mewisme/mew/internal/app",
		"github.com/mewisme/mew/internal/runner",
		"github.com/mewisme/mew/internal/transaction",
		"github.com/mewisme/mew/internal/resolver",
		"github.com/mewisme/mew/internal/linker",
		"github.com/mewisme/mew/internal/store",
		"github.com/mewisme/mew/internal/lifecycle",
	}
	presentationPath := "github.com/mewisme/mew/internal/presentation"
	for _, pkgPath := range domainNoPresentation {
		p, ok := byPath[pkgPath]
		if !ok {
			t.Fatalf("missing package %s", pkgPath)
		}
		for _, imp := range p.Imports {
			if imp == presentationPath || strings.HasPrefix(imp, presentationPath+"/") {
				t.Errorf("%s must not import %s", pkgPath, imp)
			}
		}
	}

	charmPrefixes := []string{"charm.land/", "github.com/charmbracelet/"}
	for _, p := range pkgs {
		if strings.HasPrefix(p.ImportPath, presentationPath) {
			continue
		}
		if strings.HasPrefix(p.ImportPath, "github.com/mewisme/mew/internal/cli") {
			continue
		}
		if !strings.HasPrefix(p.ImportPath, "github.com/mewisme/mew/internal/") {
			continue
		}
		for _, imp := range p.Imports {
			for _, pref := range charmPrefixes {
				if strings.HasPrefix(imp, pref) {
					t.Errorf("%s must not import Charm package %s", p.ImportPath, imp)
				}
			}
		}
	}
}

func TestInternalImportGraphAcyclic(t *testing.T) {
	root := repoRoot(t)
	pkgs := goList(t, root)
	graph := make(map[string][]string)
	for _, p := range pkgs {
		if !strings.HasPrefix(p.ImportPath, "github.com/mewisme/mew/internal/") {
			continue
		}
		var deps []string
		for _, imp := range p.Imports {
			if strings.HasPrefix(imp, "github.com/mewisme/mew/internal/") {
				deps = append(deps, imp)
			}
		}
		graph[p.ImportPath] = deps
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(graph))
	var stack []string
	var visit func(string) bool
	visit = func(n string) bool {
		color[n] = gray
		stack = append(stack, n)
		for _, d := range graph[n] {
			switch color[d] {
			case gray:
				t.Errorf("cycle involving %s -> %s (stack %v)", n, d, stack)
				return true
			case white:
				if visit(d) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = black
		return false
	}
	for n := range graph {
		if color[n] == white {
			visit(n)
		}
	}
}

func TestTransactionBoundaryDocumented(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "transaction-boundary.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, needle := range []string{"inspect", "resolve", "plan", "fetch", "verify", "stage", "validate", "commit", "rollback"} {
		if !strings.Contains(text, needle) {
			t.Errorf("transaction-boundary.md missing %q", needle)
		}
	}
}

func TestNodeAugmentationJSSurfaceDocumented(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "node-augmentation.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, needle := range []string{"stock Node", "libnode", "loader", "preload", "go:embed"} {
		if !strings.Contains(text, needle) {
			t.Errorf("node-augmentation.md missing %q", needle)
		}
	}
}

func TestDocsConsistency(t *testing.T) {
	root := repoRoot(t)

	// 1. CLAUDE.md must reference TOOLS.md.
	claudePath := filepath.Join(root, ".claude", "CLAUDE.md")
	claude, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("cannot read .claude/CLAUDE.md: %v", err)
	}
	claudeText := string(claude)
	if !strings.Contains(claudeText, "TOOLS.md") {
		t.Error(".claude/CLAUDE.md does not reference TOOLS.md")
	}

	// 2. TOOLS.md must contain a "Quick command routing" section.
	toolsPath := filepath.Join(root, "TOOLS.md")
	tools, err := os.ReadFile(toolsPath)
	if err != nil {
		t.Fatalf("cannot read TOOLS.md: %v", err)
	}
	toolsText := string(tools)
	if !strings.Contains(toolsText, "Quick command routing") {
		t.Error("TOOLS.md missing 'Quick command routing' section")
	}

	// 3. TOOLS.md must contain the canonical-source statement.
	if !strings.Contains(toolsText, "Canonical tool inventory") {
		t.Error("TOOLS.md missing 'Canonical tool inventory' statement")
	}

	// 4. Makefile must have the docs-check target.
	makePath := filepath.Join(root, "Makefile")
	makefile, err := os.ReadFile(makePath)
	if err != nil {
		t.Fatalf("cannot read Makefile: %v", err)
	}
	makeText := string(makefile)
	if !strings.Contains(makeText, "docs-check") {
		t.Error("Makefile missing docs-check target")
	}

	// 5. docs-check must be included in the quality target's dependencies.
	// Target lines start at column 0: "quality:".
	found := false
	for _, line := range strings.Split(makeText, "\n") {
		if strings.HasPrefix(line, "quality:") {
			if !strings.Contains(line, "docs-check") {
				t.Error("Makefile quality target does not include docs-check")
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("Makefile missing quality target")
	}

	// 6. Quick-command entries in TOOLS.md must reference real Makefile targets
	// or tool paths.  Extract every `make X` from the quick-command table and
	// verify X exists in the Makefile as a target.
	qcStart := strings.Index(toolsText, "## Quick command routing")
	if qcStart < 0 {
		t.Fatal("cannot find quick-command routing section")
	}
	// Scan until the next "## " heading.
	qcEnd := strings.Index(toolsText[qcStart+1:], "\n## ")
	if qcEnd < 0 {
		qcEnd = len(toolsText) - qcStart - 1
	}
	qcSection := toolsText[qcStart : qcStart+1+qcEnd]

	for _, line := range strings.Split(qcSection, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		// Third column is the command.
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		cmd := strings.TrimSpace(parts[2])
		if cmd == "Command" || cmd == "" {
			continue
		}
		if strings.HasPrefix(cmd, "`make ") {
			target := strings.TrimPrefix(cmd, "`make ")
			target = strings.TrimSuffix(target, "`")
			target = strings.TrimSpace(target)
			// Check target exists in Makefile.
			needle := "\n" + target + ":"
			if !strings.Contains(makeText, needle) {
				t.Errorf("TOOLS.md quick-command references 'make %s' but Makefile has no '%s:' target", target, target)
			}
		}
	}
}
