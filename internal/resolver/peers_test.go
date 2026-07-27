package resolver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/testkit"
)

func engineWithPackuments(t testing.TB, packs map[string]registry.Packument) (*resolver.Engine, string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		if strings.Contains(p, "/-/") {
			http.NotFound(w, r)
			return
		}
		var pack registry.Packument
		var ok bool
		for name, pk := range packs {
			if p == registry.EncodeNamePath(name) || p == name {
				pack = pk
				ok = true
				break
			}
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pack)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cacheDir := filepath.Join(t.TempDir(), "registry-cache")
	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		CacheDir:   cacheDir,
		HTTPClient: srv.Client(),
	})
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         t.TempDir(),
		GlobalPath:  filepath.Join(t.TempDir(), "missing-global.jsonc"),
		ProjectPath: filepath.Join(t.TempDir(), "missing-project.jsonc"),
		Env:         []string{},
		CLI:         map[string]any{"registry": srv.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolver.NewEngine(client, eff, project.IdentityMew), srv.URL
}

func reactPackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"react": {
			Name:     "react",
			DistTags: map[string]string{"latest": "18.2.0"},
			Versions: map[string]registry.VersionMeta{
				"18.2.0": {
					Name:    "react",
					Version: "18.2.0",
					Dist:    registry.Dist{Integrity: "sha512-react", Tarball: "react-18.2.0.tgz"},
					PeerDependencies: map[string]string{
						"react-dom": "^18.0.0",
					},
				},
			},
		},
		"react-dom": {
			Name:     "react-dom",
			DistTags: map[string]string{"latest": "18.2.0"},
			Versions: map[string]registry.VersionMeta{
				"18.2.0": {
					Name:    "react-dom",
					Version: "18.2.0",
					Dist:    registry.Dist{Integrity: "sha512-dom", Tarball: "react-dom-18.2.0.tgz"},
				},
			},
		},
	}
}

func TestResolvePeerProviders(t *testing.T) {
	eng, _ := engineWithPackuments(t, reactPackuments())
	root := writeProject(t, `{
  "name": "app",
  "version": "1.0.0",
  "dependencies": {
    "react": "^18.0.0",
    "react-dom": "^18.0.0"
  }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var reactKey string
	for _, p := range res.Graph.Packages {
		if p.ID.Name != "react" {
			continue
		}
		if len(p.ID.PeerProviderContext) == 0 {
			t.Fatalf("react missing peer providers: %#v", p.ID)
		}
		if p.ID.PeerProviderContext[0].Name != "react-dom" || p.ID.PeerProviderContext[0].Version != "18.2.0" {
			t.Fatalf("unexpected peer providers: %#v", p.ID.PeerProviderContext)
		}
		reactKey = p.ID.Key()
	}
	if reactKey == "" {
		t.Fatal("react not resolved")
	}
	if reactKey != "react@18.2.0#react-dom@18.2.0" {
		t.Fatalf("react key=%q", reactKey)
	}
	for _, d := range res.Decisions {
		if d.Package == "react" && len(d.PeerProviders) == 0 {
			t.Fatalf("decision missing peer providers: %#v", d)
		}
	}
}

func TestResolveStrictPeerMissing(t *testing.T) {
	eng, _ := engineWithPackuments(t, reactPackuments())
	root := writeProject(t, `{
  "name": "app",
  "version": "1.0.0",
  "dependencies": { "react": "^18.0.0" }
}`)
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: &policy.Policy{StrictPeerDependencies: true},
	})
	if err == nil {
		t.Fatal("expected strict peer error")
	}
	if !strings.Contains(err.Error(), "react-dom") {
		t.Fatalf("want react-dom in error: %v", err)
	}
}

func TestResolveAutoInstallPeers(t *testing.T) {
	eng, _ := engineWithPackuments(t, reactPackuments())
	root := writeProject(t, `{
  "name": "app",
  "version": "1.0.0",
  "dependencies": { "react": "^18.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: &policy.Policy{AutoInstallPeers: true, StrictPeerDependencies: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	foundDOM := false
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "react-dom" {
			foundDOM = true
		}
	}
	if !foundDOM {
		t.Fatal("react-dom should be auto-installed")
	}
}
