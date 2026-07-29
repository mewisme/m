package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/testkit"
)

func TestPublishProvenanceWithoutProviderBlocksUpload(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	putCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			putCount++
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	projDir := t.TempDir()
	testkit.CopyFixture(t, "pack/minimal-package", projDir)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	cfg := `{
  "registry": "` + srv.URL + `"
}
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Publish(ctx, ac, PublishOptions{Provenance: true})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Unsupported {
		t.Fatalf("code=%s want %s: %v", apperr.CodeOf(err), apperr.Unsupported, err)
	}
	if putCount != 0 {
		t.Fatalf("PUT count=%d want 0", putCount)
	}
}

func TestPublishProvenanceWithoutProviderDryRunFails(t *testing.T) {
	testkit.CleanEnv(t)

	projDir := t.TempDir()
	testkit.CopyFixture(t, "pack/minimal-package", projDir)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Publish(ctx, ac, PublishOptions{Provenance: true, DryRun: true})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Unsupported {
		t.Fatalf("code=%s want %s: %v", apperr.CodeOf(err), apperr.Unsupported, err)
	}
}
