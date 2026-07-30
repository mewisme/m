package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/jsonfile"
	"github.com/mewisme/mew/internal/manifest"
)

func BenchmarkDispatchBuiltinExact(b *testing.B) {
	root := NewMRoot(testBuildInfo())
	phase := PhaseAResult{Selector: "install"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveDispatch(root, phase, "", nil)
	}
}

func BenchmarkDispatchAliasExact(b *testing.B) {
	root := NewMRoot(testBuildInfo())
	phase := PhaseAResult{Selector: "i"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveDispatch(root, phase, "", nil)
	}
}

func BenchmarkDispatchScriptCachedManifest(b *testing.B) {
	projDir := b.TempDir()
	pkg := map[string]any{
		"name":    "bench",
		"version": "1.0.0",
		"scripts": map[string]string{"dev": "echo"},
	}
	raw, _ := jsonfile.Marshal(pkg)
	_ = os.WriteFile(filepath.Join(projDir, "package.json"), raw, 0o644)
	root := NewMRoot(testBuildInfo())
	eff := &config.Effective{Values: map[string]config.Value{
		"runner.direct_scripts.enabled": {Raw: true},
	}}
	phase := PhaseAResult{Selector: "dev"}
	_, _ = loadManifestScripts(projDir)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveDispatch(root, phase, projDir, eff)
	}
}

func BenchmarkDispatchSuggestSmall(b *testing.B) {
	root := NewMRoot(testBuildInfo())
	phase := PhaseAResult{Selector: "instal"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveDispatch(root, phase, b.TempDir(), nil)
	}
}

func BenchmarkDispatchSuggest1000Scripts(b *testing.B) {
	projDir := b.TempDir()
	scripts := map[string]string{}
	for i := 0; i < 1000; i++ {
		scripts[fmt.Sprintf("script%04d", i)] = "echo"
	}
	pkg := map[string]any{
		"name":    "bench",
		"version": "1.0.0",
		"scripts": scripts,
	}
	raw, _ := jsonfile.Marshal(pkg)
	_ = os.WriteFile(filepath.Join(projDir, "package.json"), raw, 0o644)
	root := NewMRoot(testBuildInfo())
	phase := PhaseAResult{Selector: "script0999"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveDispatch(root, phase, projDir, nil)
	}
}

func BenchmarkDispatchScriptColdManifest(b *testing.B) {
	// ponytail: clears in-process cache only; OS page cache may still warm reads.
	projDir := b.TempDir()
	pkg := map[string]any{
		"name":    "bench",
		"version": "1.0.0",
		"scripts": map[string]string{"dev": "echo"},
	}
	raw, _ := jsonfile.Marshal(pkg)
	_ = os.WriteFile(filepath.Join(projDir, "package.json"), raw, 0o644)
	root := NewMRoot(testBuildInfo())
	eff := &config.Effective{Values: map[string]config.Value{
		"runner.direct_scripts.enabled": {Raw: true},
	}}
	phase := PhaseAResult{Selector: "dev"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manifest.ClearCacheForTest()
		ResolveDispatch(root, phase, projDir, eff)
	}
}

func BenchmarkDispatchReservedSetBuild(b *testing.B) {
	root := NewMRoot(testBuildInfo())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reservedSetForRoot(root)
	}
}
