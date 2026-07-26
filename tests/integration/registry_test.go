package integration_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/testkit"
)

func TestRegistryClientAgainstFixture(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cacheDir := filepath.Join(t.TempDir(), "registry-cache")

	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		CacheDir:   cacheDir,
		HTTPClient: srv.Client(),
	})
	p, err := client.Packument(context.Background(), srv.URL, "lodash")
	if err != nil {
		t.Fatal(err)
	}
	if p.DistTags["latest"] != "4.17.21" {
		t.Fatalf("%v", p.DistTags)
	}
	meta, err := client.Metadata(context.Background(), "lodash", "4.17.21")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Integrity == "" || meta.TarballURL == "" {
		t.Fatalf("%+v", meta)
	}

	// Second fetch should 304
	hits := reg.HitCount()
	if _, err := client.Packument(context.Background(), srv.URL, "lodash"); err != nil {
		t.Fatal(err)
	}
	if reg.HitCount() != hits+1 {
		t.Fatalf("hits %d → %d", hits, reg.HitCount())
	}

	offline := registry.NewClient(registry.Options{
		BaseURL: srv.URL, CacheDir: cacheDir, Offline: true,
	})
	if _, err := offline.Packument(context.Background(), srv.URL, "lodash"); err != nil {
		t.Fatal(err)
	}
}
