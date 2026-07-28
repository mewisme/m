package integration_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestCleanHomeDoesNotTouchRealHome(t *testing.T) {
	realHome := os.Getenv("USERPROFILE")
	if realHome == "" {
		realHome = os.Getenv("HOME")
	}
	if realHome == "" {
		t.Skip("no real home")
	}
	marker := filepath.Join(realHome, ".mew-test-should-not-appear-0008")
	_ = os.Remove(marker)

	info := testkit.CleanEnv(t)
	probe := filepath.Join(info.Home, "wrote-in-clean-home")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("clean-home leaked marker into real home %s", marker)
	}
	if !strings.HasPrefix(info.CacheDir, info.Home) {
		t.Fatalf("cache not under home: %s", info.CacheDir)
	}
}

func TestFixtureRegistrySmoke(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	client := &http.Client{Transport: http.DefaultTransport}
	resp, err := client.Get(srv.URL + "/lodash")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("packument status %d", resp.StatusCode)
	}

	resp2, err := client.Get(srv.URL + "/lodash/-/lodash-4.17.21.tgz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp2.Body.Close() }()
	data, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	want, err := reg.BlobSHA256("tarballs/lodash-4.17.21.tgz")
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("integrity mismatch got %s want %s", got, want)
	}
}

func TestCopyProjectFixture(t *testing.T) {
	home := testkit.TempHome(t)
	projects := []string{
		"projects/basic-cjs",
		"projects/basic-esm",
		"projects/typescript-app",
		"projects/workspace-simple",
	}
	for _, rel := range projects {
		t.Run(rel, func(t *testing.T) {
			dest := filepath.Join(home, filepath.FromSlash(rel))
			testkit.CopyFixture(t, rel, dest)
			if _, err := os.Stat(filepath.Join(dest, "package.json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEvilArchiveFixturePresent(t *testing.T) {
	dir := testkit.FixtureDir(t, "security/evil-archives")
	b, err := os.ReadFile(filepath.Join(dir, "path-traversal-members.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("../../etc/passwd")) {
		t.Fatal("missing traversal member")
	}
}
