package app_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/testkit"
)

type authRecord struct {
	mu         sync.Mutex
	packuments []string
	tarballs   []string
}

func (r *authRecord) record(req *http.Request) {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if strings.Contains(req.URL.Path, "/-/") {
		r.tarballs = append(r.tarballs, auth)
	} else {
		r.packuments = append(r.packuments, auth)
	}
}

func startRecordingRegistry(t *testing.T) (*httptest.Server, *authRecord) {
	t.Helper()
	reg := testkit.LoadRegistry(t, "registry/v1")
	inner := reg.Start(t)
	rec := &authRecord{}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rec.record(req)
		u, err := url.Parse(inner.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
		req.RequestURI = ""
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	t.Cleanup(proxy.Close)
	return proxy, rec
}

func authTestContext(t *testing.T, srvURL string, env []string) *app.Context {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("NPM_TOKEN", "host-token")
	root := t.TempDir()
	cfgPath := filepath.Join(root, "m.jsonc")
	registryLine := `{"registry":"` + srvURL + `","registry.auth_token_env":"NPM_TOKEN"}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"auth-test","version":"1.0.0","dependencies":{"pkg-a":"^1.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(registryLine+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := app.Options{CWD: root, ConfigPath: cfgPath}
	if env != nil {
		opts.Env = env
	} else {
		opts.Env = []string{"NPM_TOKEN=invocation-token"}
	}
	ac, err := app.New(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	return ac
}

func TestResolverPackumentUsesSnapshotAuth(t *testing.T) {
	srv, rec := startRecordingRegistry(t)
	ac := authTestContext(t, srv.URL, []string{"NPM_TOKEN=invocation-token"})
	proj, err := app.OpenProject(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Resolve(context.Background(), proj.Root, resolver.ResolveOptions{}); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.packuments) == 0 {
		t.Fatal("expected packument requests with auth")
	}
	for _, auth := range rec.packuments {
		if auth != "Bearer invocation-token" {
			t.Fatalf("packument auth=%q want Bearer invocation-token", auth)
		}
	}
}

func TestTarballDownloadUsesSnapshotAuth(t *testing.T) {
	srv, rec := startRecordingRegistry(t)
	ac := authTestContext(t, srv.URL, []string{"NPM_TOKEN=invocation-token"})
	if _, err := app.Add(context.Background(), ac, "pkg-a", app.AddOptions{}); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.tarballs) == 0 {
		t.Fatal("expected tarball requests with auth")
	}
	for _, auth := range rec.tarballs {
		if auth != "Bearer invocation-token" {
			t.Fatalf("tarball auth=%q want Bearer invocation-token", auth)
		}
	}
}

func TestAuthStableAfterAmbientEnvChange(t *testing.T) {
	srv, rec := startRecordingRegistry(t)
	ac := authTestContext(t, srv.URL, []string{"NPM_TOKEN=invocation-token"})
	t.Setenv("NPM_TOKEN", "changed-host-token")
	proj, err := app.OpenProject(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Resolve(context.Background(), proj.Root, resolver.ResolveOptions{}); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, auth := range rec.packuments {
		if auth != "Bearer invocation-token" {
			t.Fatalf("packument auth=%q want Bearer invocation-token", auth)
		}
	}
}

func TestEmptyEnvNoBearerHeader(t *testing.T) {
	srv, rec := startRecordingRegistry(t)
	ac := authTestContext(t, srv.URL, []string{})
	proj, err := app.OpenProject(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Resolve(context.Background(), proj.Root, resolver.ResolveOptions{}); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.packuments) > 0 || len(rec.tarballs) > 0 {
		t.Fatalf("expected no auth headers, packuments=%v tarballs=%v", rec.packuments, rec.tarballs)
	}
}

func TestScopedRegistryUsesSnapshotAuth(t *testing.T) {
	srv, rec := startRecordingRegistry(t)
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("NPM_TOKEN", "host-token")
	root := t.TempDir()
	cfgPath := filepath.Join(root, "m.jsonc")
	scopeURL := srv.URL
	cfg := `{"registry":"` + srv.URL + `","registries":{"@scope":"` + scopeURL + `"},"registry.auth_token_env":"NPM_TOKEN"}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"scoped-auth","version":"1.0.0","dependencies":{"@scope/pkg":"^1.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(cfg+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ac, err := app.New(context.Background(), app.Options{
		CWD: root, ConfigPath: cfgPath, Env: []string{"NPM_TOKEN=scoped-invocation"},
	})
	if err != nil {
		t.Fatal(err)
	}
	proj := &project.Project{Root: root, Identity: project.IdentityMew}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{}); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.packuments) == 0 {
		t.Fatal("expected scoped packument request")
	}
	for _, auth := range rec.packuments {
		if auth != "Bearer scoped-invocation" {
			t.Fatalf("packument auth=%q want Bearer scoped-invocation", auth)
		}
	}
}
