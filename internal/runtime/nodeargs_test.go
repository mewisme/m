package runtime

import (
	"testing"
)

func TestParseNodeArgs(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantV8         []string
		wantEntrypoint string
		wantApp        []string
		wantErr        bool
	}{
		{
			name:           "simple entrypoint",
			args:           []string{"app.js"},
			wantV8:         nil,
			wantEntrypoint: "app.js",
			wantApp:        nil,
		},
		{
			name:           "with leading dash dash",
			args:           []string{"--", "app.js"},
			wantV8:         nil,
			wantEntrypoint: "app.js",
			wantApp:        nil,
		},
		{
			name:           "with V8 flags",
			args:           []string{"--trace-warnings", "app.js"},
			wantV8:         []string{"--trace-warnings"},
			wantEntrypoint: "app.js",
			wantApp:        nil,
		},
		{
			name:           "value-taking flag next arg",
			args:           []string{"--max-old-space-size", "4096", "server.mjs"},
			wantV8:         []string{"--max-old-space-size", "4096"},
			wantEntrypoint: "server.mjs",
			wantApp:        nil,
		},
		{
			name:           "value-taking flag equals form",
			args:           []string{"--max-old-space-size=4096", "server.mjs"},
			wantV8:         []string{"--max-old-space-size=4096"},
			wantEntrypoint: "server.mjs",
			wantApp:        nil,
		},
		{
			name:           "with app args after dash dash",
			args:           []string{"app.js", "--", "--port", "3000"},
			wantV8:         nil,
			wantEntrypoint: "app.js",
			wantApp:        []string{"--port", "3000"},
		},
		{
			name:           "with V8 flags and app args",
			args:           []string{"--trace-warnings", "--max-old-space-size=4096", "server.mjs", "--", "--port", "3000"},
			wantV8:         []string{"--trace-warnings", "--max-old-space-size=4096"},
			wantEntrypoint: "server.mjs",
			wantApp:        []string{"--port", "3000"},
		},
		{
			name:    "empty args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "no entrypoint, only flags",
			args:    []string{"--trace-warnings"},
			wantErr: true,
		},
		{
			name:    "flag missing value",
			args:    []string{"--max-old-space-size"},
			wantErr: true,
		},
		{
			name:           "stdin dash entrypoint",
			args:           []string{"-"},
			wantV8:         nil,
			wantEntrypoint: "-",
			wantApp:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v8, ep, app, err := ParseNodeArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strSliceEq(v8, tt.wantV8) {
				t.Errorf("v8Args = %v, want %v", v8, tt.wantV8)
			}
			if ep != tt.wantEntrypoint {
				t.Errorf("entrypoint = %q, want %q", ep, tt.wantEntrypoint)
			}
			if !strSliceEq(app, tt.wantApp) {
				t.Errorf("appArgs = %v, want %v", app, tt.wantApp)
			}
		})
	}
}

func TestBuildArgv(t *testing.T) {
	plan := &LaunchPlan{
		NodeExe: "/usr/bin/node",
		PreloadAssets: []PreloadAsset{
			{Path: "/cache/preload.cjs", ModuleType: "cjs"},
			{Path: "/cache/preload.mjs", ModuleType: "esm"},
		},
		Entrypoint: "app.js",
		AppArgs:    []string{"--port", "3000"},
	}

	argv := BuildArgv(plan, []string{"--trace-warnings"})

	// Expected: node --trace-warnings --require /cache/preload.cjs --import /cache/preload.mjs app.js --port 3000
	expected := []string{
		"/usr/bin/node",
		"--trace-warnings",
		"--require", "/cache/preload.cjs",
		"--import", "/cache/preload.mjs",
		"app.js",
		"--port", "3000",
	}

	if !strSliceEq(argv, expected) {
		t.Errorf("argv = %v, want %v", argv, expected)
	}
}

func TestBuildArgvZeroAugmentation(t *testing.T) {
	plan := &LaunchPlan{
		NodeExe:          "/usr/bin/node",
		PreloadAssets:    []PreloadAsset{{Path: "/cache/preload.cjs", ModuleType: "cjs"}},
		Entrypoint:       "app.js",
		AppArgs:          []string{"--port", "3000"},
		ZeroAugmentation: true,
	}

	argv := BuildArgv(plan, nil)

	expected := []string{
		"/usr/bin/node",
		"app.js",
		"--port", "3000",
	}

	if !strSliceEq(argv, expected) {
		t.Errorf("argv = %v, want %v", argv, expected)
	}
}

func TestParseNodeArgsWithDoubleDashBeforeEntrypoint(t *testing.T) {
	// m node-args -- -- app.js
	// Cobra strips the first --, so args = ["--", "app.js"]
	v8, ep, app, err := ParseNodeArgs([]string{"--", "app.js"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep != "app.js" {
		t.Errorf("entrypoint = %q, want %q", ep, "app.js")
	}
	if len(v8) != 0 {
		t.Errorf("v8 = %v, want empty", v8)
	}
	if len(app) != 0 {
		t.Errorf("app = %v, want empty", app)
	}
}


func TestParseNodeArgsEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantV8         []string
		wantEntrypoint string
		wantApp        []string
		wantErr        bool
	}{
		{
			name:           "entrypoint starting with dash via explicit path",
			args:           []string{"./-app.js"},
			wantV8:         nil,
			wantEntrypoint: "./-app.js",
			wantApp:        nil,
		},
		{
			name:           "double dash before entrypoint with app args",
			args:           []string{"--", "app.js", "--port", "3000"},
			wantV8:         nil,
			wantEntrypoint: "app.js",
			wantApp:        []string{"--port", "3000"},
		},
		{
			name:           "node flag with equals form",
			args:           []string{"--max-old-space-size=4096", "app.js"},
			wantV8:         []string{"--max-old-space-size=4096"},
			wantEntrypoint: "app.js",
			wantApp:        nil,
		},
		{
			name:           "multiple node flags with mixed forms",
			args:           []string{"--inspect", "--max-old-space-size=4096", "app.js", "arg1"},
			wantV8:         []string{"--inspect", "--max-old-space-size=4096"},
			wantEntrypoint: "app.js",
			wantApp:        []string{"arg1"},
		},
		{
			name:           "node flag requiring value",
			args:           []string{"--require", "./polyfill.js", "app.js"},
			wantV8:         []string{"--require", "./polyfill.js"},
			wantEntrypoint: "app.js",
			wantApp:        nil,
		},
		{
			name:           "unknown flag after entrypoint treated as app arg",
			args:           []string{"app.js", "--unknown-flag"},
			wantV8:         nil,
			wantEntrypoint: "app.js",
			wantApp:        []string{"--unknown-flag"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v8, entry, app, err := ParseNodeArgs(tc.args)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strSliceEq(v8, tc.wantV8) {
				t.Fatalf("v8=%v, want %v", v8, tc.wantV8)
			}
			if entry != tc.wantEntrypoint {
				t.Fatalf("entry=%q, want %q", entry, tc.wantEntrypoint)
			}
			if !strSliceEq(app, tc.wantApp) {
				t.Fatalf("app=%v, want %v", app, tc.wantApp)
			}
		})
	}
}

func strSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
