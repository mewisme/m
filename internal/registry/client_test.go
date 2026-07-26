package registry_test

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/diagnostics"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/testkit"
)

func TestPackumentFetchAndCache304(t *testing.T) {
	reg := testkit.LoadRegistry(t, "")
	srv := reg.Start(t)
	cacheDir := t.TempDir()
	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		CacheDir:   cacheDir,
		HTTPClient: srv.Client(),
	})

	p1, err := client.Packument(context.Background(), srv.URL, "lodash")
	if err != nil {
		t.Fatal(err)
	}
	if p1.Name != "lodash" {
		t.Fatalf("%+v", p1)
	}
	hits1 := reg.HitCount()

	p2, err := client.Packument(context.Background(), srv.URL, "lodash")
	if err != nil {
		t.Fatal(err)
	}
	if p2.DistTags["latest"] != "4.17.21" {
		t.Fatalf("%v", p2.DistTags)
	}
	hits2 := reg.HitCount()
	if hits2 != hits1+1 {
		t.Fatalf("expected conditional revalidate hit, hits %d → %d", hits1, hits2)
	}

	meta, err := client.Metadata(context.Background(), "lodash", "latest")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != "4.17.21" || meta.Integrity == "" {
		t.Fatalf("%+v", meta)
	}
}

func TestOfflineMiss(t *testing.T) {
	client := registry.NewClient(registry.Options{
		BaseURL:  "http://127.0.0.1:1",
		CacheDir: t.TempDir(),
		Offline:  true,
	})
	_, err := client.Packument(context.Background(), client.BaseURL(), "lodash")
	if err == nil {
		t.Fatal("expected offline miss")
	}
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestOfflineHit(t *testing.T) {
	reg := testkit.LoadRegistry(t, "")
	srv := reg.Start(t)
	cacheDir := t.TempDir()
	online := registry.NewClient(registry.Options{
		BaseURL: srv.URL, CacheDir: cacheDir, HTTPClient: srv.Client(),
	})
	if _, err := online.Packument(context.Background(), srv.URL, "lodash"); err != nil {
		t.Fatal(err)
	}
	offline := registry.NewClient(registry.Options{
		BaseURL: srv.URL, CacheDir: cacheDir, Offline: true,
	})
	p, err := offline.Packument(context.Background(), srv.URL, "lodash")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "lodash" {
		t.Fatal(p.Name)
	}
}

func TestScopedPackument(t *testing.T) {
	reg := testkit.LoadRegistry(t, "")
	srv := reg.Start(t)
	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		CacheDir:   t.TempDir(),
		HTTPClient: srv.Client(),
	})
	p, err := client.Packument(context.Background(), srv.URL, "@scope/pkg")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "@scope/pkg" {
		t.Fatalf("%q", p.Name)
	}
}

func TestAuthNeverLogged(t *testing.T) {
	token := "supersecrettokenvalue999"
	var errBuf strings.Builder
	rep := diagnostics.NewReporter(diagnostics.Options{
		Out: ioDiscard{}, Err: &errBuf, Format: "default", Color: diagnostics.ColorNever,
	})
	reg := testkit.LoadRegistry(t, "")
	atomic.StoreInt32(&reg.ForceStatus, http.StatusUnauthorized)
	srv := reg.Start(t)
	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		CacheDir:   t.TempDir(),
		HTTPClient: srv.Client(),
		AuthToken:  token,
	})
	_, err := client.Packument(context.Background(), srv.URL, "lodash")
	if err == nil {
		t.Fatal("expected auth error")
	}
	rep.Error(err)
	out := errBuf.String() + err.Error()
	if strings.Contains(out, token) {
		t.Fatalf("token leaked: %q", out)
	}
}

func TestWorkerPoolBound(t *testing.T) {
	reg := testkit.LoadRegistry(t, "")
	srv := reg.Start(t)
	var concurrent atomic.Int32
	var maxSeen atomic.Int32
	transport := &countingTransport{
		base: srv.Client().Transport,
		onDo: func() {
			n := concurrent.Add(1)
			for {
				cur := maxSeen.Load()
				if n <= cur || maxSeen.CompareAndSwap(cur, n) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			concurrent.Add(-1)
		},
	}
	hc := &http.Client{Transport: transport}
	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		CacheDir:   t.TempDir(),
		HTTPClient: hc,
		MaxWorkers: 2,
	})
	names := []string{"lodash", "@scope/pkg", "lodash", "@scope/pkg"}
	if _, err := client.Packuments(context.Background(), srv.URL, names); err != nil {
		t.Fatal(err)
	}
	if maxSeen.Load() > 2 {
		t.Fatalf("max concurrent=%d", maxSeen.Load())
	}
}

func TestResolveScopeConfig(t *testing.T) {
	home := testkit.TempHome(t)
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD: home,
		CLI: map[string]any{
			"registry":          "https://registry.npmjs.org",
			"registries.@scope": "http://scoped.example",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := registry.ResolveRegistryURL("@scope/pkg", eff, "", project.IdentityMew)
	if got != "http://scoped.example" {
		t.Fatalf("%s", got)
	}
}

func TestNotFound(t *testing.T) {
	reg := testkit.LoadRegistry(t, "")
	srv := reg.Start(t)
	client := registry.NewClient(registry.Options{
		BaseURL: srv.URL, CacheDir: t.TempDir(), HTTPClient: srv.Client(),
	})
	_, err := client.Packument(context.Background(), srv.URL, "no-such-pkg")
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

type countingTransport struct {
	base http.RoundTripper
	onDo func()
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if c.onDo != nil {
		c.onDo()
	}
	base := c.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
