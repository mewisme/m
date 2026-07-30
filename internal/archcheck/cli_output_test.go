package archcheck_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cliDirectPrintAllowlist files exempt from direct fmt.Fprint* checks.
var cliDirectPrintAllowlist = map[string]bool{
	"static_output.go":    true,
	"completion.go":       true,
	"dispatch.go":         true,
	"execute.go":          true,
	"explain_cmd.go":      true, // resolver owns colored explanation output until UX-0007
	"registry_cmd.go":     true, // single-value path queries (cache dir, view)
	"config_cmd.go":       true, // config get single value stdout
	"store_cmd.go":        true, // store path single value stdout
	"sbom_cmd.go":         true, // machine SBOM document on stdout
	"snapshot_format.go":  true,
	"lock_diff_format.go": true,
	// ponytail: pending migration after UX-0003 static command groups
	"bench_cmd.go":              true,
	"conformance_cmd.go":        true,
	"conformance_runner_cmd.go": true,
	"development.go":            true,
	"env_cmd.go":                true,
	"fetch_cmd.go":              true,
	"lock_cmd.go":               true,
	"mx_cache_cmd.go":           true,
	"patch_cmd.go":              true,
	"project_cmd.go":            true,
	"publish_cmd.go":            true,
	"resolve_cmd.go":            true,
	"root.go":                   true,
	"trust_cmd.go":              true,
}

func TestCLINoUnallowlistedDirectPrint(t *testing.T) {
	root := repoRoot(t)
	cliDir := filepath.Join(root, "internal", "cli")
	fset := token.NewFileSet()
	var offenders []string
	err := filepath.Walk(cliDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		base := filepath.Base(path)
		if cliDirectPrintAllowlist[base] {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "fmt" {
				return true
			}
			switch sel.Sel.Name {
			case "Fprint", "Fprintln", "Fprintf":
			default:
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			if isOutOrErrWriter(call.Args[0]) {
				pos := fset.Position(call.Pos())
				offenders = append(offenders, pos.String())
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Fatalf("direct stdout/stderr prints outside allowlist:\n%s", strings.Join(offenders, "\n"))
	}
}

func isOutOrErrWriter(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	cmd, ok := sel.X.(*ast.Ident)
	if !ok || cmd.Name != "cmd" {
		return false
	}
	switch sel.Sel.Name {
	case "OutOrStdout", "ErrOrStderr":
		return true
	default:
		return false
	}
}
