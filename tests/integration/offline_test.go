package integration_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func TestOfflineInstallFromFullCache(t *testing.T) {
	env := testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := copyOfflineFixture(t)
	cfgPath := writeRegistryConfig(t, projDir, srv.URL)
	seedOfflineCache(t, env.CacheDir, srv.URL, srv.Client(), reg.Root)

	code, out := runM(t, projDir, cfgPath, "install", "--offline")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	for _, pkg := range []string{"pkg-a", "pkg-b", "pkg-c"} {
		if _, err := os.Stat(filepath.Join(projDir, "node_modules", pkg, "package.json")); err != nil {
			t.Fatalf("%s: %v", pkg, err)
		}
	}
}

func TestOfflineInstallFailsWithMissingList(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := copyOfflineFixture(t)
	cfgPath := writeRegistryConfig(t, projDir, srv.URL)

	code, out := runM(t, projDir, cfgPath, "install", "--offline")
	if code == 0 {
		t.Fatalf("expected failure, out=%s", out)
	}
}

func TestOfflineWarmFasterThanCold(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	coldProj := copyOfflineFixture(t)
	coldCfg := writeRegistryConfig(t, coldProj, srv.URL)

	coldStart := time.Now()
	code, coldOut := runM(t, coldProj, coldCfg, "install")
	if code != 0 {
		t.Fatalf("cold install exit=%d out=%s", code, coldOut)
	}
	coldElapsed := time.Since(coldStart)

	warmProj := copyOfflineFixture(t)
	warmCfg := writeRegistryConfig(t, warmProj, srv.URL)
	warmStart := time.Now()
	warmCode, warmOut := runM(t, warmProj, warmCfg, "install", "--offline")
	warmElapsed := time.Since(warmStart)
	if warmCode != 0 {
		t.Fatalf("warm offline exit=%d out=%s", warmCode, warmOut)
	}

	const minColdWall = 100 * time.Millisecond
	if coldElapsed < minColdWall {
		t.Skipf("cold install %s below %s; wall-clock comparison unreliable on this runner", coldElapsed, minColdWall)
	}
	if warmElapsed >= coldElapsed {
		t.Fatalf("warm wall %s not faster than cold %s", warmElapsed, coldElapsed)
	}
}

func copyOfflineFixture(t *testing.T) string {
	t.Helper()
	src := filepath.Join(testkit.ModuleRoot(t), "testdata", "offline", "full-cache-project")
	dst := t.TempDir()
	copyTreeIntegration(t, src, dst)
	return dst
}

func writeRegistryConfig(t *testing.T, projDir, registryURL string) string {
	t.Helper()
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+registryURL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func copyTreeIntegration(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		from := filepath.Join(src, ent.Name())
		to := filepath.Join(dst, ent.Name())
		if ent.IsDir() {
			if err := os.MkdirAll(to, 0o755); err != nil {
				t.Fatal(err)
			}
			copyTreeIntegration(t, from, to)
			continue
		}
		in, err := os.Open(from)
		if err != nil {
			t.Fatal(err)
		}
		out, err := os.Create(to)
		if err != nil {
			_ = in.Close()
			t.Fatal(err)
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = in.Close()
			_ = out.Close()
			t.Fatal(err)
		}
		_ = in.Close()
		_ = out.Close()
	}
}

func seedOfflineCache(t *testing.T, cacheDir, registryURL string, hc *http.Client, regRoot string) {
	t.Helper()
	ctx := context.Background()
	client := registry.NewClient(registry.Options{
		BaseURL:    registryURL,
		CacheDir:   filepath.Join(cacheDir, "registry"),
		HTTPClient: hc,
	})
	for _, name := range []string{"pkg-a", "pkg-b", "pkg-c"} {
		if _, err := client.Packument(ctx, registryURL, name); err != nil {
			t.Fatalf("packument %s: %v", name, err)
		}
	}
	blobRoot := filepath.Join(cacheDir, "blobs")
	for _, spec := range []struct {
		rel, integrity string
	}{
		{"tarballs/pkg-a-1.0.0.tgz", "sha256-2e1afab8b566a6ac1019ae2ba9201ea8a036b0ca1463ed2b22673d4cc87b2354"},
		{"tarballs/pkg-b-1.2.0.tgz", "sha256-24c92f24878bdc43d34f0397c47ebd529869e7e444c28813ecfda1b8a52d2d65"},
		{"tarballs/pkg-c-1.0.1.tgz", "sha256-9b75a3d4e441037d52cbcf35988036a38e8e931c5d77c71d729abc85a89ded6a"},
	} {
		data, err := os.ReadFile(filepath.Join(regRoot, spec.rel))
		if err != nil {
			t.Fatal(err)
		}
		hex := spec.integrity[len("sha256-"):]
		dst := filepath.Join(blobRoot, "sha256", hex)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
