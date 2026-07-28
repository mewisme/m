package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/store"
	"github.com/mewisme/mew/internal/testkit"
)

func TestFetchLodashFromFixtureRegistry(t *testing.T) {
	info := testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	cache := filepath.Join(info.CacheDir, "blobs")
	_ = os.MkdirAll(cache, 0o755)

	ctx := context.Background()
	eff := &config.Effective{Values: map[string]config.Value{
		"cache.dir": {Raw: info.CacheDir},
	}}
	ac := &app.Context{Config: eff, Ctx: ctx, CWD: t.TempDir()}

	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	plan := app.FetchPlan{Packages: []app.FetchPackage{{
		Name: "lodash", Version: "4.17.21",
		TarballURL: srv.URL + "/lodash/-/lodash-4.17.21.tgz",
		Integrity:  integrity,
	}}}
	dest := filepath.Join(t.TempDir(), "out")
	results, err := app.Fetch(ctx, ac, plan, dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "lodash" {
		t.Fatalf("%+v", results)
	}
	if _, err := os.Stat(filepath.Join(results[0].Dest, "package.json")); err != nil {
		t.Fatal(err)
	}
}

func TestFetchIntegrityMismatchAborts(t *testing.T) {
	home := testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	ac := &app.Context{Config: &config.Effective{Values: map[string]config.Value{
		"cache.dir": {Raw: home.CacheDir},
	}}, Ctx: context.Background(), CWD: t.TempDir()}
	dest := filepath.Join(t.TempDir(), "out")
	_, err := app.Fetch(context.Background(), ac, app.FetchPlan{Packages: []app.FetchPackage{{
		Name: "lodash", Version: "4.17.21",
		TarballURL: srv.URL + "/lodash/-/lodash-4.17.21.tgz",
		Integrity:  "sha256-0000000000000000000000000000000000000000000000000000000000000000",
	}}}, dest)
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
	if entries, _ := os.ReadDir(dest); len(entries) != 0 {
		t.Fatalf("dest should be empty: %v", entries)
	}
}

func TestFetchOfflineCacheHit(t *testing.T) {
	home := testkit.CleanEnv(t)
	root := testkit.ModuleRoot(t)
	seed := filepath.Join(root, "testdata", "fetch", "offline-cache-hit")
	blobSrc := filepath.Join(seed, "blobs", "sha256", "758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63")
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	data, err := os.ReadFile(tgz)
	if err != nil {
		t.Fatal(err)
	}
	dstBlob := filepath.Join(home.CacheDir, "blobs", "sha256", "758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63")
	if err := os.MkdirAll(filepath.Dir(dstBlob), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(blobSrc); err == nil {
		data, err = os.ReadFile(blobSrc)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(dstBlob, data, 0o644); err != nil {
		t.Fatal(err)
	}

	ac := &app.Context{Config: &config.Effective{Values: map[string]config.Value{
		"cache.dir": {Raw: home.CacheDir},
		"offline":   {Raw: true},
	}}, Ctx: context.Background(), CWD: t.TempDir()}
	dest := filepath.Join(t.TempDir(), "out")
	results, err := app.Fetch(context.Background(), ac, app.FetchPlan{Packages: []app.FetchPackage{{
		Name: "lodash", Version: "4.17.21",
		TarballURL: "http://offline.invalid/lodash.tgz",
		Integrity:  "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63",
	}}}, dest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(results[0].Dest, "package.json")); err != nil {
		t.Fatal(err)
	}
}

func TestCacheVerifyCLI(t *testing.T) {
	home := testkit.CleanEnv(t)
	st := store.NewDir(filepath.Join(home.CacheDir, "blobs"))
	key := store.Key("sha256/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	if err := st.Put(context.Background(), key, nil); err != nil {
		t.Fatal(err)
	}
	ac := &app.Context{Config: &config.Effective{Values: map[string]config.Value{
		"cache.dir": {Raw: home.CacheDir},
	}}, Ctx: context.Background()}
	res, err := app.VerifyBlobCache(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK != 1 || res.Bad != 0 {
		t.Fatalf("%+v", res)
	}
}

func TestFetchPlanJSONRoundTrip(t *testing.T) {
	plan := app.FetchPlan{Packages: []app.FetchPackage{{
		Name: "lodash", Version: "4.17.21", TarballURL: "http://x/t.tgz",
		Integrity: "sha256-abc",
	}}}
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	var back app.FetchPlan
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Packages[0].Name != "lodash" {
		t.Fatalf("%+v", back)
	}
}

func TestCorruptHashFixtureRejectedByDownloader(t *testing.T) {
	tgz := filepath.Join(testkit.FixtureDir(t, "archives"), "corrupt-hash.tgz")
	raw, err := os.ReadFile(tgz)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(raw)
	}))
	defer srv.Close()
	st := store.NewDir(filepath.Join(t.TempDir(), "blobs"))
	dl := &fetch.Downloader{Client: srv.Client(), Store: st, StagingDir: t.TempDir()}
	_, err = dl.Download(context.Background(), fetch.DownloadRequest{
		URL: srv.URL, Integrity: "sha256-0000000000000000000000000000000000000000000000000000000000000000",
	})
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
}

func TestArchiveExtractRejectsTraversalIntegration(t *testing.T) {
	tgz := filepath.Join(testkit.FixtureDir(t, "archives"), "traversal-attack.tgz")
	dest := t.TempDir()
	err := archive.Extract(context.Background(), tgz, dest, archive.DefaultOptions())
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("%v", err)
	}
}
