package runner_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/workspace"
)

var benchScripts = map[string]string{
	"dev":     "vite",
	"build":   "tsc",
	"test":    "node test.js",
	"test:a":  "node a.js",
	"test:b":  "node b.js",
	"predev":  "echo pre",
	"postdev": "echo post",
}

func BenchmarkLookupExact(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := runner.Lookup(benchScripts, "dev"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLookupRegex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := runner.Lookup(benchScripts, "/^test:/"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExpandPlans(b *testing.B) {
	names := []string{"dev"}
	for i := 0; i < b.N; i++ {
		if plans := runner.ExpandPlans(benchScripts, names); len(plans) != 1 {
			b.Fatalf("got %d plans", len(plans))
		}
	}
}

func BenchmarkBuildEnv(b *testing.B) {
	host := []string{"HOME=/tmp/home", "PATH=/usr/bin:/bin"}
	opts := runner.EnvOptions{
		HostEnv:     host,
		InitCWD:     "/proj",
		PackageDir:  "/proj",
		NodeModules: "/proj/node_modules",
		PackageJSON: "/proj/package.json",
		PackageName: "bench-pkg",
		PackageVer:  "1.0.0",
		Event:       "dev",
		Script:      "vite",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runner.BuildEnv(opts)
	}
}

func BenchmarkScheduleWideDAG(b *testing.B) {
	modRoot := moduleRootBench(b)
	g, err := workspace.BuildGraph(filepath.Join(modRoot, "fixtures", "workspace-runner", "large"))
	if err != nil {
		b.Fatal(err)
	}
	paths, err := runner.SelectMembers(g, true, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := runner.BuildSchedule(g, paths, runner.OrderTopological, 4); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPrefixWriter(b *testing.B) {
	var out bytes.Buffer
	w := runner.NewPrefixWriter("pkg", "build", "stdout", &out, nil, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = w.Write([]byte("line\n"))
	}
}

func moduleRootBench(b *testing.B) string {
	b.Helper()
	dir, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			b.Fatal("go.mod not found")
		}
		dir = parent
	}
}
