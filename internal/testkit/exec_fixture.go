package testkit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/linker"
)

// SetupExecFixture creates a minimal project with one verified local bin.
func SetupExecFixture(t testing.TB, command string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"exec-fixture","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return setupExecBinAtRoot(t, root, command)
}

// SetupExecWorkspaceFixture creates a minimal workspace with one local bin at the root importer.
func SetupExecWorkspaceFixture(t testing.TB, command string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"exec-ws","private":true,"workspaces":["packages/*"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte("packages:\n  - packages/*\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(root, "packages", "api")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"api","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return setupExecBinAtRoot(t, root, command)
}

func setupExecBinAtRoot(t testing.TB, root, command string) string {
	t.Helper()
	nm := filepath.Join(root, "node_modules")
	pkgDir := filepath.Join(nm, "demo-pkg")
	binDir := filepath.Join(nm, ".bin")
	for _, d := range []string{nm, pkgDir, binDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script := filepath.Join(pkgDir, "cli.js")
	body := []byte("console.log(process.argv.slice(2).join(' '));\n")
	if err := os.WriteFile(script, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linker.WriteBins(nm, []linker.BinSource{{
		Cmd: command, Target: "cli.js", PackageDir: pkgDir, NodeModules: nm,
	}}); err != nil {
		t.Fatal(err)
	}
	doc, err := binmeta.BuildDocument(binmeta.PublishInput{
		NodeModules: nm, GenerationID: "test-gen", ImporterIdentity: ".", LayoutMode: binmeta.LayoutHoisted,
		Sources: []linker.BinSource{{Cmd: command, Target: "cli.js", PackageDir: pkgDir}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := binmeta.Write(nm, doc); err != nil {
		t.Fatal(err)
	}
	if err := writeExecGenerationBinding(root, "test-gen", doc.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if node := os.Getenv("MEW_NODE"); node == "" && runtime.GOOS != "windows" {
		if p, err := execLookNode(); err == nil {
			t.Setenv("MEW_TRUSTED_NODE", p)
		}
	}
	return root
}

func writeExecGenerationBinding(root, generationID, fingerprint string) error {
	genDir := filepath.Join(root, ".mew")
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		return err
	}
	genBody, err := json.MarshalIndent(map[string]string{
		"generationID": generationID,
		"fingerprint":  fingerprint,
	}, "", "  ")
	if err != nil {
		return err
	}
	genBody = append(genBody, '\n')
	return os.WriteFile(filepath.Join(genDir, "generation.json"), genBody, 0o644)
}

func execLookNode() (string, error) {
	candidates := []string{"/usr/local/bin/node", "/opt/homebrew/bin/node"}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", os.ErrNotExist
}

// EnableDirectBinDispatch turns on MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH.
func EnableDirectBinDispatch(t testing.TB) {
	t.Helper()
	t.Setenv("MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH", "1")
}
