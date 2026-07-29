package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestPublishDryRunNoPUT(t *testing.T) {
	projDir, cfgPath, _ := setupPublishProject(t)
	code, out := runM(t, projDir, cfgPath, "publish", "--dry-run")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "dry-run") {
		t.Fatalf("missing dry-run marker: %q", out)
	}
	reg := currentPublishRegistry(t)
	if reg.PublishCount() != 0 {
		t.Fatalf("expected no PUT, got %d", reg.PublishCount())
	}
}

func TestPublishPUTObserved(t *testing.T) {
	projDir, cfgPath, _ := setupPublishProject(t)
	code, out := runM(t, projDir, cfgPath, "publish", "--otp", "123456")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "minimal-pack-fixture@1.2.3") {
		t.Fatalf("stdout %q", out)
	}
	reg := currentPublishRegistry(t)
	if reg.PublishCount() != 1 {
		t.Fatalf("publish count %d", reg.PublishCount())
	}
	pub := reg.Publishes()[0]
	if pub.OTP != "123456" {
		t.Fatalf("otp %q", pub.OTP)
	}
	if !strings.HasPrefix(pub.Auth, "Bearer ") {
		t.Fatalf("auth %q", pub.Auth)
	}
	var body map[string]any
	if err := json.Unmarshal(pub.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "minimal-pack-fixture" {
		t.Fatalf("%v", body["name"])
	}
	attach, _ := body["_attachments"].(map[string]any)
	if len(attach) != 1 {
		t.Fatalf("attachments %v", attach)
	}
}

func setupPublishProject(t *testing.T) (projDir, cfgPath, srvURL string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")
	t.Setenv("NPM_TOKEN", "npm_test_token_for_publish")

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	publishRegistryHolder = reg
	t.Cleanup(func() { publishRegistryHolder = nil })

	projDir = t.TempDir()
	testkit.CopyFixture(t, "pack/minimal-package", projDir)
	cfgPath = filepath.Join(projDir, "m.jsonc")
	cfg := `{
  "registry": "` + srv.URL + `",
  "registry.auth_token_env": "NPM_TOKEN"
}
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return projDir, cfgPath, srv.URL
}

var publishRegistryHolder *testkit.FixtureRegistry

func currentPublishRegistry(t *testing.T) *testkit.FixtureRegistry {
	t.Helper()
	if publishRegistryHolder == nil {
		t.Fatal("publish registry not set")
	}
	return publishRegistryHolder
}
