package resolver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/resolver"
)

func TestExplainPeerGolden(t *testing.T) {
	eng, _ := engineWithPackuments(t, reactPackuments())
	root := writeProject(t, `{
  "name": "app",
  "version": "1.0.0",
  "dependencies": { "react": "^18.0.0" }
}`)
	tree, err := eng.ExplainPeer(context.Background(), root, "react-dom", resolver.ResolveOptions{
		Policy: &policy.Policy{StrictPeerDependencies: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tree == nil {
		t.Fatal("expected conflict tree")
	}

	goldenDir := filepath.Join("..", "..", "testdata", "resolver", "explain")
	jsonGolden := filepath.Join(goldenDir, "react-missing-react-dom.json")
	txtGolden := filepath.Join(goldenDir, "react-missing-react-dom.txt")

	gotJSON, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	gotJSON = append(gotJSON, '\n')
	gotHuman := resolver.FormatConflictTree(*tree)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(goldenDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(jsonGolden, gotJSON, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(txtGolden, []byte(gotHuman), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("updated explain goldens")
	}

	wantJSON, err := os.ReadFile(jsonGolden)
	if err != nil {
		t.Fatalf("read json golden (set UPDATE_GOLDEN=1): %v", err)
	}
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("json golden mismatch\n--- got ---\n%s\n--- want ---\n%s", gotJSON, wantJSON)
	}

	wantHuman, err := os.ReadFile(txtGolden)
	if err != nil {
		t.Fatalf("read txt golden (set UPDATE_GOLDEN=1): %v", err)
	}
	if string(wantHuman) != gotHuman {
		t.Fatalf("human golden mismatch\n--- got ---\n%s\n--- want ---\n%s", gotHuman, wantHuman)
	}
}

func TestBuildPeerConflictTreeExpanded(t *testing.T) {
	conf := resolver.PeerConflict{
		Package:           "react",
		Peer:              "react-dom",
		Range:             "^18.0.0",
		Importer:          "root",
		SearchPath:        []string{"root", "react@18.2.0"},
		AutoInstallPolicy: false,
	}
	tree := resolver.BuildPeerConflictTree(conf, nil)
	if tree.Root.RequiringPackage != "react" {
		t.Fatalf("requiring=%q", tree.Root.RequiringPackage)
	}
	if tree.Root.Importer != "root" {
		t.Fatalf("importer=%q", tree.Root.Importer)
	}
	if len(tree.Root.SearchPath) != 2 {
		t.Fatalf("searchPath=%v", tree.Root.SearchPath)
	}
	if tree.Root.Remediation == "" {
		t.Fatal("expected remediation")
	}
	if !strings.Contains(tree.Root.Remediation, "react-dom") {
		t.Fatalf("remediation=%q", tree.Root.Remediation)
	}
}
