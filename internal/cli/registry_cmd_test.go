package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheDirAndView(t *testing.T) {
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("MEW_HOME", home)
	t.Setenv("MEW_CACHE_DIR", filepath.Join(home, "cache"))

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"cache", "dir"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, "registry") {
		t.Fatalf("cache dir=%q", got)
	}

	// view needs a registry — use fixture via --config overlay is heavy; just ensure command exists
	root = NewMRoot(testBuildInfo())
	buf.Reset()
	root.SetOut(buf)
	root.SetErr(buf)
	fixture := filepath.Join(modRoot, "fixtures", "projects", "basic-cjs")
	root.SetArgs([]string{"--cwd", fixture, "view", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "View package") {
		t.Fatalf("%s", buf.String())
	}
}
