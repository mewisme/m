package app

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func TestPreflightOfflineReportsMissingBlobAndPackument(t *testing.T) {
	testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := copyOfflineProject(t)
	cfgPath := writeOfflineConfig(t, projDir, srv.URL)

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Offline: true, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		t.Fatal(err)
	}
	g, err := readLockHints(ctx, ac, proj)
	if err != nil || g == nil {
		t.Fatalf("graph: %v", err)
	}

	report, err := PreflightOffline(ctx, ac, proj, g, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatal("expected missing prerequisites")
	}
	kinds := map[OfflineMissingKind]int{}
	for _, m := range report.Missing {
		kinds[m.Kind]++
	}
	if kinds[OfflineMissingPackument] == 0 {
		t.Fatalf("expected packument misses: %+v", report.Missing)
	}
	if kinds[OfflineMissingBlob] == 0 {
		t.Fatalf("expected blob misses: %+v", report.Missing)
	}
	err = offlinePreflightError(report)
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("got %v", err)
	}
	msg := err.Error()
	for _, want := range []string{"pkg-a", "packument", "blob"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q: %q", want, msg)
		}
	}
}

func TestPreflightOfflineOKWithSeededCache(t *testing.T) {
	env := testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := copyOfflineProject(t)
	cfgPath := writeOfflineConfig(t, projDir, srv.URL)
	seedRegistryAndBlobs(t, env.CacheDir, srv.URL, srv.Client(), reg.Root)

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Offline: true, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		t.Fatal(err)
	}
	g, err := readLockHints(ctx, ac, proj)
	if err != nil || g == nil {
		t.Fatalf("graph: %v", err)
	}

	report, err := PreflightOffline(ctx, ac, proj, g, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("missing=%+v", report.Missing)
	}
}

func TestPreflightOfflineCorruptBlobQuarantined(t *testing.T) {
	env := testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := copyOfflineProject(t)
	cfgPath := writeOfflineConfig(t, projDir, srv.URL)
	seedRegistryAndBlobs(t, env.CacheDir, srv.URL, srv.Client(), reg.Root)

	blobRoot := filepath.Join(env.CacheDir, "blobs")
	corruptHex := "2e1afab8b566a6ac1019ae2ba9201ea8a036b0ca1463ed2b22673d4cc87b2354"
	corruptPath := filepath.Join(blobRoot, "sha256", corruptHex)
	if err := os.WriteFile(corruptPath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Offline: true, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		t.Fatal(err)
	}
	g, err := readLockHints(ctx, ac, proj)
	if err != nil || g == nil {
		t.Fatalf("graph: %v", err)
	}

	report, err := PreflightOffline(ctx, ac, proj, g, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatal("expected corrupt blob to fail preflight")
	}
	foundBlob := false
	for _, m := range report.Missing {
		if m.Kind == OfflineMissingBlob {
			foundBlob = true
		}
	}
	if !foundBlob {
		t.Fatalf("missing=%+v", report.Missing)
	}
	quarantine := filepath.Join(blobRoot, ".quarantine", "sha256", corruptHex)
	if _, err := os.Stat(quarantine); err != nil {
		t.Fatalf("expected quarantine: %v", err)
	}
}

func copyOfflineProject(t *testing.T) string {
	t.Helper()
	src := filepath.Join(testkit.ModuleRoot(t), "testdata", "offline", "full-cache-project")
	dst := t.TempDir()
	copyOfflineTree(t, src, dst)
	return dst
}

func writeOfflineConfig(t *testing.T, projDir, registryURL string) string {
	t.Helper()
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+registryURL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func copyOfflineTree(t *testing.T, src, dst string) {
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
			copyOfflineTree(t, from, to)
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

func seedRegistryAndBlobs(t *testing.T, cacheDir, registryURL string, hc *http.Client, regRoot string) {
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
