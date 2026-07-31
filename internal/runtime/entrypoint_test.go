package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsJSFile(t *testing.T) {
	tests := []struct {
		selector string
		want     bool
	}{
		{"app.js", true},
		{"App.JS", true}, // case insensitive
		{"server.mjs", true},
		{"loader.cjs", true},
		{"src/app.js", true},       // dir separator
		{"src\\app.js", true},      // backslash
		{"readme.md", false},       // unsupported ext
		{"Makefile", false},        // no ext
		{"", false},                // empty
		{"build", false},           // bare name
		{"./script.ts", true},      // dir separator even with .ts
		{"/abs/path/file", true},   // absolute path
	}
	for _, tt := range tests {
		got := IsJSFile(tt.selector)
		if got != tt.want {
			t.Errorf("IsJSFile(%q) = %v, want %v", tt.selector, got, tt.want)
		}
	}
}

func TestResolveEntrypoint(t *testing.T) {
	tmp := t.TempDir()

	// create a valid JS file
	jsPath := filepath.Join(tmp, "app.js")
	if err := os.WriteFile(jsPath, []byte("console.log(1);"), 0o644); err != nil {
		t.Fatal(err)
	}

	// create a .mjs file
	mjsPath := filepath.Join(tmp, "server.mjs")
	if err := os.WriteFile(mjsPath, []byte("console.log(2);"), 0o644); err != nil {
		t.Fatal(err)
	}

	// create a .cjs file
	cjsPath := filepath.Join(tmp, "loader.cjs")
	if err := os.WriteFile(cjsPath, []byte("module.exports = {};"), 0o644); err != nil {
		t.Fatal(err)
	}

	// create a directory
	subDir := filepath.Join(tmp, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		cwd       string
		selector  string
		wantOK    bool
		wantEmpty bool
	}{
		{"absolute path", "/", jsPath, true, false},
		{"relative path", tmp, "app.js", true, false},
		{"mjs file", tmp, "server.mjs", true, false},
		{"cjs file", tmp, "loader.cjs", true, false},
		{"missing file", tmp, "missing.js", false, false},
		{"directory", tmp, "subdir", false, false},
		{"unsupported ext", tmp, "readme.md", false, false},
		{"empty selector", tmp, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveEntrypoint(tt.cwd, tt.selector)
			if tt.wantOK && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantOK {
				if tt.wantEmpty && got != "" {
					t.Errorf("expected empty result, got %q", got)
				}
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if got == "" {
				t.Fatal("expected non-empty result")
			}
		})
	}
}

func TestResolveEntrypointWithSpaces(t *testing.T) {
	tmp := t.TempDir()
	spacePath := filepath.Join(tmp, "my app.js")
	if err := os.WriteFile(spacePath, []byte("console.log(1);"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveEntrypoint(tmp, "my app.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != spacePath {
		t.Fatalf("got %q, want %q", got, spacePath)
	}
}
