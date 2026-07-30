package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func startSubsetRegistry(t *testing.T, packuments ...string) *httptest.Server {
	t.Helper()
	root := testkit.FixtureDir(t, "registry/v1")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p := strings.TrimPrefix(req.URL.Path, "/")
		if strings.Contains(p, "/-/") {
			parts := strings.Split(p, "/-/")
			if len(parts) != 2 {
				http.NotFound(w, req)
				return
			}
			http.ServeFile(w, req, filepath.Join(root, "tarballs", parts[1]))
			return
		}
		key := "packuments/" + p + ".json"
		abs := filepath.Join(root, filepath.FromSlash(key))
		if _, err := os.Stat(abs); err != nil {
			http.NotFound(w, req)
			return
		}
		http.ServeFile(w, req, abs)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestRepeatedAddPublishNodeModules(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	srv := startSubsetRegistry(t, "lodash", "pkg-a")
	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "mewjs-test",
  "version": "1.0.0",
  "private": true
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "add", "lodash")
	if code != 0 {
		t.Fatalf("first add exit=%d out=%s", code, out)
	}
	code, out = runM(t, projDir, cfgPath, "add", "pkg-a")
	if code != 0 {
		t.Fatalf("second add exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash")); err != nil {
		t.Fatalf("lodash missing after second add: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "pkg-a")); err != nil {
		t.Fatalf("pkg-a missing after second add: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules.mew-old")); err == nil {
		t.Fatal("stale node_modules.mew-old")
	}
	_ = context.Background()
}
