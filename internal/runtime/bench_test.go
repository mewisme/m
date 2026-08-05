package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/node"
	"github.com/mewisme/mew/internal/runtime"
	"github.com/mewisme/mew/internal/testkit"
)

func BenchmarkAssetExtractionCold(b *testing.B) {
	for b.Loop() {
		b.StopTimer()
		info := testkit.CleanEnv(b)
		eff := configFromCacheDir(b, info.CacheDir)
		b.StartTimer()
		_, err := runtime.EnsureAssets(eff)
		if err != nil {
			b.Fatal(err)
		}
		if err := runtime.VerifyCache(eff); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssetVerifyWarm(b *testing.B) {
	info := testkit.CleanEnv(b)
	eff := configFromCacheDir(b, info.CacheDir)
	if _, err := runtime.EnsureAssets(eff); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		if err := runtime.VerifyCache(eff); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPlanJS(b *testing.B) {
	b.StopTimer()
	info := testkit.CleanEnv(b)
	eff := configFromCacheDir(b, info.CacheDir)
	dir := b.TempDir()
	jsFile := filepath.Join(dir, "app.js")
	if err := os.WriteFile(jsFile, []byte("console.log(1);"), 0o644); err != nil {
		b.Fatal(err)
	}
	nodeInst, err := node.Discover(context.Background(), node.Request{WorkingDir: dir})
	if err != nil {
		b.Skip("node required: " + err.Error())
	}
	_ = nodeInst
	b.StartTimer()
	for b.Loop() {
		_, err := runtime.Plan(context.Background(), runtime.LaunchRequest{
			Entrypoint:       jsFile,
			WorkingDir:       dir,
			AugmentationMode: runtime.AugmentDefault,
			Stdio: runtime.LaunchStdio{
				Stdin:  nil,
				Stdout: nil,
				Stderr: nil,
			},
		}, eff)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildArgv(b *testing.B) {
	plan := &runtime.LaunchPlan{
		NodeExe:          "/usr/bin/node",
		NodeVersion:      "22.0.0",
		NodeCapabilities: []string{"require-preload", "import-preload", "module-register", "source-maps"},
		PreloadAssets: []runtime.PreloadAsset{
			{Path: "/cache/preload.cjs", ModuleType: "cjs"},
			{Path: "/cache/preload.mjs", ModuleType: "esm"},
			{Path: "/cache/loader-register.mjs", ModuleType: "esm"},
		},
		Entrypoint:       "/proj/app.js",
		AppArgs:          []string{"--port", "3000"},
		ZeroAugmentation: false,
	}
	v8Args := []string{"--trace-warnings", "--max-old-space-size=4096"}
	for b.Loop() {
		_ = runtime.BuildArgv(plan, v8Args)
	}
}

func configFromCacheDir(b *testing.B, cacheDir string) *config.Effective {
	b.Helper()
	return &config.Effective{
		Values: map[string]config.Value{
			"cache.dir": {Raw: cacheDir},
		},
	}
}
