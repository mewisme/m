package runner_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/runner"
)

func envMap(vars []string) map[string]string {
	out := make(map[string]string, len(vars))
	for _, kv := range vars {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return out
}

func TestBuildEnvNpmLifecycleVars(t *testing.T) {
	host := []string{"HOME=/tmp/home", "PATH=/usr/bin"}
	opts := runner.EnvOptions{
		HostEnv:     host,
		InitCWD:     "/proj",
		PackageDir:  "/proj/pkg",
		NodeModules: "/proj/node_modules",
		PackageJSON: "/proj/pkg/package.json",
		PackageName: "demo",
		PackageVer:  "1.2.3",
		Event:       "dev",
		Script:      "vite",
	}
	got := runner.BuildEnv(opts)
	if got.Dir != "/proj/pkg" {
		t.Fatalf("dir %q, want /proj/pkg", got.Dir)
	}
	m := envMap(got.Vars)
	checks := map[string]string{
		"INIT_CWD":             "/proj",
		"npm_lifecycle_event":  "dev",
		"npm_lifecycle_script": "vite",
		"npm_package_name":     "demo",
		"npm_package_version":  "1.2.3",
		"npm_package_json":     "/proj/pkg/package.json",
		"HOME":                 "/tmp/home",
	}
	pathKey := "PATH"
	if runtime.GOOS == "windows" {
		pathKey = "Path"
	}
	for key, want := range checks {
		if got, ok := m[key]; !ok || got != want {
			t.Fatalf("%s=%q, want %q (missing=%v)", key, got, want, !ok)
		}
	}
	pathVal, ok := m[pathKey]
	if !ok {
		t.Fatalf("missing %s", pathKey)
	}
	wantBin := "/proj/node_modules/.bin"
	if runtime.GOOS == "windows" {
		wantBin = strings.ReplaceAll(wantBin, "/", `\`)
	}
	if !strings.HasPrefix(pathVal, wantBin) {
		t.Fatalf("%s=%q, want prefix %q", pathKey, pathVal, wantBin)
	}
	if !strings.Contains(pathVal, "/usr/bin") && !strings.Contains(pathVal, `\usr\bin`) {
		t.Fatalf("%s=%q, want host PATH preserved", pathKey, pathVal)
	}
}

func TestBuildEnvPreservesHostWithoutRestrictedEmpty(t *testing.T) {
	host := []string{"CUSTOM=value", "npm_package_name=old", "PATH=/bin"}
	got := runner.BuildEnv(runner.EnvOptions{
		HostEnv:    host,
		InitCWD:    "/proj",
		PackageDir: "/proj",
		Event:      "dev",
		Script:     "echo hi",
	})
	m := envMap(got.Vars)
	if m["CUSTOM"] != "value" {
		t.Fatalf("CUSTOM=%q, want value", m["CUSTOM"])
	}
	if m["npm_package_name"] == "old" {
		t.Fatalf("npm_package_name=%q, want host value overridden", m["npm_package_name"])
	}
	pathKey := "PATH"
	if runtime.GOOS == "windows" {
		pathKey = "Path"
	}
	if _, ok := m[pathKey]; !ok {
		t.Fatalf("missing %s", pathKey)
	}
}

func TestPackageJSONPath(t *testing.T) {
	got := runner.PackageJSONPath("/proj/pkg")
	want := "/proj/pkg/package.json"
	if runtime.GOOS == "windows" {
		want = `\proj\pkg\package.json`
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
