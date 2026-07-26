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
		"github.com/mewisme/m/internal/app": true,
		"github.com/mewisme/m/internal/cli": true,
	}

	for _, cmdPath := range []string{"github.com/mewisme/m/cmd/m", "github.com/mewisme/m/cmd/mx"} {
		p, ok := byPath[cmdPath]
		if !ok {
			t.Fatalf("missing package %s", cmdPath)
		}
		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, "github.com/mewisme/m/") {
				continue
			}
			if !allowedCmd[imp] {
				t.Errorf("%s must not import %s (only internal/app and internal/cli)", cmdPath, imp)
			}
		}
	}

	cliForbidden := []string{
		"github.com/mewisme/m/internal/linker",
		"github.com/mewisme/m/internal/store",
		"github.com/mewisme/m/internal/fetch",
	}
	cli := byPath["github.com/mewisme/m/internal/cli"]
	for _, bad := range cliForbidden {
		for _, imp := range cli.Imports {
			if imp == bad || strings.HasPrefix(imp, bad+"/") {
				t.Errorf("internal/cli must not import %s", imp)
			}
		}
	}

	resolverForbidden := []string{
		"github.com/mewisme/m/internal/linker",
		"github.com/mewisme/m/internal/transaction",
		"github.com/mewisme/m/internal/runner",
		"github.com/mewisme/m/internal/fetch",
		"github.com/mewisme/m/internal/store",
	}
	res := byPath["github.com/mewisme/m/internal/resolver"]
	for _, bad := range resolverForbidden {
		for _, imp := range res.Imports {
			if imp == bad || strings.HasPrefix(imp, bad+"/") {
				t.Errorf("internal/resolver must not import %s", imp)
			}
		}
	}

	diagForbidden := []string{
		"github.com/mewisme/m/internal/registry",
		"github.com/mewisme/m/internal/fetch",
		"github.com/mewisme/m/internal/linker",
	}
	for _, pkgPath := range []string{
		"github.com/mewisme/m/internal/apperr",
		"github.com/mewisme/m/internal/diagnostics",
		"github.com/mewisme/m/internal/trace",
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
		"github.com/mewisme/m/internal/resolver",
		"github.com/mewisme/m/internal/linker",
		"github.com/mewisme/m/internal/fetch",
	}
	for _, pkgPath := range []string{"github.com/mewisme/m/internal/config", "github.com/mewisme/m/internal/project"} {
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
		"github.com/mewisme/m/internal/fetch",
		"github.com/mewisme/m/internal/linker",
		"github.com/mewisme/m/internal/registry",
	}
	for _, pkgPath := range []string{
		"github.com/mewisme/m/internal/graph",
		"github.com/mewisme/m/internal/plan",
		"github.com/mewisme/m/internal/snapshot",
		"github.com/mewisme/m/internal/manifest",
		"github.com/mewisme/m/internal/policy",
		"github.com/mewisme/m/internal/capsule",
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
}

func TestInternalImportGraphAcyclic(t *testing.T) {
	root := repoRoot(t)
	pkgs := goList(t, root)
	graph := make(map[string][]string)
	for _, p := range pkgs {
		if !strings.HasPrefix(p.ImportPath, "github.com/mewisme/m/internal/") {
			continue
		}
		var deps []string
		for _, imp := range p.Imports {
			if strings.HasPrefix(imp, "github.com/mewisme/m/internal/") {
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
