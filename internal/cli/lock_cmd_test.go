package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile/mlock"
)

func TestLockValidateAndFormat(t *testing.T) {
	g := loadTestGraph(t, "simple-app.json")
	specs := mlock.SpecifiersFromGraph(g)
	doc, err := mlock.FromGraph(g, specs, mlock.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	data, err := mlock.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}

	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "app",
  "version": "1.0.0",
  "dependencies": {
    "left-pad": "^1.0.0",
    "ms": "^2.0.0"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "m.lock"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cliRoot := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	cliRoot.SetOut(buf)
	cliRoot.SetErr(buf)
	cliRoot.SetArgs([]string{"--cwd", projDir, "lock", "validate"})
	if code := ExecuteWithContext(cliRoot, context.Background()); code != 0 {
		t.Fatalf("validate exit=%d out=%s", code, buf.String())
	}

	buf.Reset()
	cliRoot.SetArgs([]string{"--cwd", projDir, "lock", "format", "--json"})
	if code := ExecuteWithContext(cliRoot, context.Background()); code != 0 {
		t.Fatalf("format exit=%d out=%s", code, buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"changed"`)) {
		t.Fatalf("format json: %s", buf.String())
	}
}

func TestLockValidateFrozenDrift(t *testing.T) {
	g := loadTestGraph(t, "simple-app.json")
	specs := mlock.SpecifiersFromGraph(g)
	doc, err := mlock.FromGraph(g, specs, mlock.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	data, err := mlock.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}

	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "app",
  "version": "1.0.0",
  "dependencies": {
    "left-pad": "^9.9.9"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "m.lock"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cliRoot := NewMRoot(testBuildInfo())
	cliRoot.SetOut(ioDiscard{})
	cliRoot.SetErr(ioDiscard{})
	cliRoot.SetArgs([]string{"--cwd", projDir, "lock", "validate", "--frozen"})
	err = cliRoot.Execute()
	if err == nil {
		t.Fatal("expected frozen drift error")
	}
	if apperr.CodeOf(err) != apperr.Lockfile {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func moduleRoot(t testing.TB) string {
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

func loadTestGraph(t testing.TB, name string) *graph.Graph {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(moduleRoot(t), "testdata", "graph", name))
	if err != nil {
		t.Fatal(err)
	}
	g, err := graph.DecodeJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	return g
}
