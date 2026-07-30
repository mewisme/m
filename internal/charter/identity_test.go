package charter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type productIdentity struct {
	SchemaVersion  int    `json:"schemaVersion"`
	FullName       string `json:"full_name"`
	ShortName      string `json:"short_name"`
	PrimaryBinary  string `json:"primary_binary"`
	PrimaryAlias   string `json:"primary_alias"`
	ExecutorBinary string `json:"executor_binary"`
	ExecutorAlias  string `json:"executor_alias"`
	NativeLockfile string `json:"native_lockfile"`
}

func loadIdentity(t *testing.T, root string) productIdentity {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "product", "identity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var id productIdentity
	if err := json.Unmarshal(b, &id); err != nil {
		t.Fatal(err)
	}
	if id.SchemaVersion != 1 {
		t.Fatalf("identity schemaVersion=%d want 1", id.SchemaVersion)
	}
	return id
}

func TestProductIdentityMatchesDocs(t *testing.T) {
	root := repoRoot(t)
	id := loadIdentity(t, root)

	readme := readDoc(t, root, "README.md")
	naming := readDoc(t, root, filepath.Join("docs", "naming.md"))
	planREADME := readDoc(t, root, filepath.Join("plans", "0000-README.md"))

	for _, doc := range []struct {
		name string
		text string
	}{
		{"README.md", readme},
		{"docs/naming.md", naming},
		{"plans/0000-README.md", planREADME},
	} {
		if !strings.Contains(doc.text, id.FullName) {
			t.Errorf("%s missing full product name %q", doc.name, id.FullName)
		}
		if !strings.Contains(doc.text, id.ShortName) {
			t.Errorf("%s missing short name %q", doc.name, id.ShortName)
		}
		for _, term := range []string{
			"`" + id.PrimaryBinary + "`",
			"`" + id.PrimaryAlias + "`",
			"`" + id.ExecutorBinary + "`",
			"`" + id.ExecutorAlias + "`",
			"`" + id.NativeLockfile + "`",
		} {
			if !strings.Contains(doc.text, term) {
				t.Errorf("%s missing %s", doc.name, term)
			}
		}
	}

	manifestRaw, err := os.ReadFile(filepath.Join(root, "plans", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Product struct {
			FullName       string `json:"full_name"`
			ShortName      string `json:"short_name"`
			Binary         string `json:"binary"`
			PrimaryAlias   string `json:"primary_alias"`
			ExecutorBinary string `json:"executor_binary"`
			ExecutorAlias  string `json:"executor_alias"`
			NativeLockfile string `json:"native_lockfile"`
		} `json:"product"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	p := manifest.Product
	if p.FullName != id.FullName || p.ShortName != id.ShortName ||
		p.Binary != id.PrimaryBinary || p.PrimaryAlias != id.PrimaryAlias ||
		p.ExecutorBinary != id.ExecutorBinary || p.ExecutorAlias != id.ExecutorAlias ||
		p.NativeLockfile != id.NativeLockfile {
		t.Fatalf("plans/manifest.json product block mismatch: %+v vs identity %+v", p, id)
	}
}

func TestCLIHelpUsesMewJSIdentity(t *testing.T) {
	root := repoRoot(t)
	id := loadIdentity(t, root)

	cliRoot := readDoc(t, root, filepath.Join("internal", "cli", "root.go"))
	if !strings.Contains(cliRoot, id.FullName) {
		t.Errorf("internal/cli/root.go missing %q in help strings", id.FullName)
	}
	if !strings.Contains(cliRoot, id.ShortName) {
		t.Errorf("internal/cli/root.go missing short name %q", id.ShortName)
	}

	// Aliases are naming-contract targets; installer distribution is deferred (README).
	readme := readDoc(t, root, "README.md")
	if !strings.Contains(readme, "Installer-shipped aliases") || !strings.Contains(readme, "not distributed automatically") {
		t.Error("README must document that mew/mewx aliases are not auto-installed")
	}
}
