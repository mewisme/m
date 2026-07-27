package testkit_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/mewisme/m/internal/testkit"
)

func TestCleanEnvSetsMEW(t *testing.T) {
	info := testkit.CleanEnv(t)
	if got := os.Getenv("MEW_HOME"); got != info.Home {
		t.Fatalf("MEW_HOME=%q", got)
	}
	if got := os.Getenv("MEW_CACHE_DIR"); got != info.CacheDir {
		t.Fatalf("MEW_CACHE_DIR=%q", got)
	}
}

func TestLoadRegistryChecksums(t *testing.T) {
	reg := testkit.LoadRegistry(t, "registry/v1")
	if len(reg.Manifest.Blobs) < 2 {
		t.Fatal("expected blobs")
	}
}

func TestRegistryServePackumentAndTarball(t *testing.T) {
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	resp, err := http.Get(srv.URL + "/lodash")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(`"name": "lodash"`)) && !bytes.Contains(body, []byte(`"name":"lodash"`)) {
		t.Fatalf("packument body: %s", body)
	}

	tarballURL := srv.URL + "/lodash/-/lodash-4.17.21.tgz"
	resp2, err := http.Get(tarballURL)
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
		t.Fatalf("tarball sha %s want %s", got, want)
	}
}

func TestNormalizeAndDiffReport(t *testing.T) {
	out := testkit.NormalizeOutput("ok C:\\Users\\x\\y\r\n2026-01-02T03:04:05Z")
	if strings.Contains(out, "Users") || strings.Contains(out, "2026") {
		t.Fatalf("not normalized: %q", out)
	}
	path := filepath.Join(t.TempDir(), "report.json")
	r := &testkit.DiffReport{
		SchemaVersion: testkit.DiffReportSchemaVersion,
		Skipped:       true,
		SkipReason:    "npm not found",
		Diffs:         []testkit.DiffItem{},
	}
	if err := testkit.WriteDiffReport(path, r); err != nil {
		t.Fatal(err)
	}
	got, err := testkit.ReadDiffReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Skipped || got.SkipReason == "" {
		t.Fatalf("%+v", got)
	}
}

func TestFaultyRoundTripperAndLimitedWriter(t *testing.T) {
	ft := &testkit.FaultyRoundTripper{After: 0}
	_, err := ft.RoundTrip(&http.Request{URL: nil})
	if err == nil {
		t.Fatal("expected network cut")
	}
	lw := &testkit.LimitedWriter{N: 3}
	_, err = lw.Write([]byte("abcdef"))
	if err == nil {
		t.Fatal("expected ENOSPC")
	}
	var pe *os.PathError
	if !errors.As(err, &pe) || !errors.Is(pe.Err, syscall.ENOSPC) {
		t.Fatalf("want ENOSPC got %v", err)
	}
}

func TestProbeFS(t *testing.T) {
	caps := testkit.ProbeFS(t, t.TempDir())
	_ = caps // platform-dependent; must not panic
}
